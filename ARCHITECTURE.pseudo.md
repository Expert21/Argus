# Argus - Architecture Draft (Pseudocode)
# Language: TBD (Python | Rust | Go)
# Purpose: Unified Linux Log Viewer

## ═══════════════════════════════════════════════════════════════════
## CORE DATA STRUCTURES
## ═══════════════════════════════════════════════════════════════════

```pseudo
struct LogEntry {
    timestamp: DateTime
    source: String          # "journald", "auth.log", "nginx/access.log"
    level: Enum(DEBUG, INFO, WARN, ERROR, CRITICAL, UNKNOWN)
    message: String
    raw: String             # Original unparsed line
    metadata: Map<String, Any>  # Extra fields (PID, unit name, etc.)
}

struct LogSource {
    id: UUID
    name: String            # Human-readable: "Nginx Access"
    source_type: Enum(JOURNALD, FILE, DIRECTORY)
    path: String?           # null for journald
    enabled: Boolean
    color: String?          # Optional custom highlight color
    filter_regex: String?   # Optional pre-filter
}

struct AppConfig {
    sources: List<LogSource>
    theme: String
    max_buffer_size: Integer    # How many log lines to keep in memory
    scroll_on_new: Boolean
}
```

## ═══════════════════════════════════════════════════════════════════
## COMPONENT ARCHITECTURE
## ═══════════════════════════════════════════════════════════════════

```
┌─────────────────────────────────────────────────────────────────────┐
│                           ARGUS TUI                                 │
│  ┌──────────────┐  ┌─────────────────────────────────────────────┐  │
│  │   Sidebar    │  │              Main Log View                  │  │
│  │              │  │                                             │  │
│  │ ▸ System     │  │  [2024-01-18 13:40:01] [nginx] GET /api...  │  │
│  │   ▸ Journal  │  │  [2024-01-18 13:40:02] [sshd] Accepted...   │  │
│  │   ▸ Kernel   │  │  [2024-01-18 13:40:03] [sudo] isaiah...     │  │
│  │ ▸ Auth       │  │  [2024-01-18 13:40:05] [ERROR] Failed...    │  │
│  │ ▸ Custom     │  │                                             │  │
│  │   ▸ Nginx    │  │                                             │  │
│  │   ▸ Docker   │  │                                             │  │
│  │              │  │                                             │  │
│  │ [+ Add New]  │  │                                             │  │
│  └──────────────┘  └─────────────────────────────────────────────┘  │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ Status: ● Live | Sources: 5 | Events: 1,247 | Filter: none    ││
│  └─────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────┘
```

## ═══════════════════════════════════════════════════════════════════
## MODULE BREAKDOWN
## ═══════════════════════════════════════════════════════════════════

### 1. CONFIG MANAGER
```pseudo
module ConfigManager {
    CONFIG_PATH = "~/.config/argus/config.yaml"
    
    function load_config() -> AppConfig:
        if file_exists(CONFIG_PATH):
            return parse_yaml(read_file(CONFIG_PATH))
        else:
            return default_config()
    
    function save_config(config: AppConfig):
        write_file(CONFIG_PATH, to_yaml(config))
    
    function add_source(source: LogSource):
        config = load_config()
        config.sources.append(source)
        save_config(config)
        emit_event(SOURCE_ADDED, source)
    
    function remove_source(source_id: UUID):
        config = load_config()
        config.sources.remove_by_id(source_id)
        save_config(config)
        emit_event(SOURCE_REMOVED, source_id)
}
```

### 2. INGESTORS (Log Source Readers)
```pseudo
trait Ingestor {
    function start() -> AsyncStream<LogEntry>
    function stop()
    function is_healthy() -> Boolean
}

class JournaldIngestor implements Ingestor {
    process: Subprocess?
    filters: List<String>  # e.g., ["-u", "nginx", "-u", "sshd"]
    
    function start() -> AsyncStream<LogEntry>:
        # Spawn: journalctl -o json -f --no-pager {filters}
        self.process = spawn_subprocess(
            "journalctl", ["-o", "json", "-f", "--no-pager"] + self.filters
        )
        
        async for line in self.process.stdout:
            json_data = parse_json(line)
            yield LogEntry {
                timestamp: parse_timestamp(json_data["__REALTIME_TIMESTAMP"]),
                source: json_data.get("_SYSTEMD_UNIT", "journal"),
                level: map_priority_to_level(json_data["PRIORITY"]),
                message: json_data["MESSAGE"],
                raw: line,
                metadata: json_data
            }
    
    function stop():
        if self.process:
            self.process.terminate()
}

class FileIngestor implements Ingestor {
    path: String
    watcher: FileWatcher?
    
    function start() -> AsyncStream<LogEntry>:
        # Use inotify/kqueue to watch for file changes
        self.watcher = watch_file(self.path)
        
        # Initial read of last N lines (optional)
        for line in tail(self.path, lines=100):
            yield parse_log_line(line)
        
        # Watch for new lines
        async for event in self.watcher:
            if event.type == MODIFIED:
                for new_line in read_new_lines(self.path):
                    yield parse_log_line(new_line)
    
    function parse_log_line(line: String) -> LogEntry:
        # Attempt to parse common log formats
        # - syslog format
        # - nginx combined
        # - apache common
        # - RFC 3339 timestamps
        # Fall back to raw if unparseable
        ...
}

class DirectoryIngestor implements Ingestor {
    # Watches all *.log files in a directory
    # Handles log rotation gracefully
    path: String
    glob_pattern: String = "*.log"
    file_ingestors: Map<String, FileIngestor>
    
    function start() -> AsyncStream<LogEntry>:
        # Watch for new files matching pattern
        # Spawn FileIngestor for each
        # Merge all streams
        ...
}
```

### 3. AGGREGATOR (Central Event Bus)
```pseudo
class Aggregator {
    buffer: RingBuffer<LogEntry>  # Fixed size, oldest dropped
    subscribers: List<Channel>
    sources: Map<UUID, Ingestor>
    
    function add_source(source: LogSource):
        ingestor = match source.source_type:
            JOURNALD -> JournaldIngestor(source)
            FILE     -> FileIngestor(source.path)
            DIRECTORY -> DirectoryIngestor(source.path)
        
        self.sources[source.id] = ingestor
        spawn_task(self.consume_ingestor(source.id, ingestor))
    
    async function consume_ingestor(id: UUID, ingestor: Ingestor):
        try:
            async for entry in ingestor.start():
                self.buffer.push(entry)
                for subscriber in self.subscribers:
                    subscriber.send(entry)
        except Error as e:
            emit_event(SOURCE_ERROR, id, e)
    
    function subscribe() -> Channel<LogEntry>:
        channel = new Channel()
        self.subscribers.append(channel)
        return channel
    
    function get_history(count: Integer) -> List<LogEntry>:
        return self.buffer.last(count)
}
```

### 4. FILTER ENGINE
```pseudo
class FilterEngine {
    active_filters: List<Filter>
    
    struct Filter {
        field: Enum(SOURCE, LEVEL, MESSAGE, ANY)
        operator: Enum(EQUALS, CONTAINS, REGEX, NOT)
        value: String
    }
    
    function apply(entry: LogEntry) -> Boolean:
        for filter in self.active_filters:
            if not matches(entry, filter):
                return false
        return true
    
    function set_level_filter(min_level: LogLevel):
        # Only show WARN and above, etc.
        ...
    
    function set_source_filter(sources: List<String>):
        # Only show specific sources
        ...
    
    function set_search(query: String):
        # Regex or fuzzy search on message
        ...
}
```

### 5. TUI APPLICATION
```pseudo
class ArgusApp {
    aggregator: Aggregator
    filter_engine: FilterEngine
    config: AppConfig
    
    # UI Components
    sidebar: SourceTreeWidget
    log_view: LogViewWidget
    status_bar: StatusBarWidget
    
    function on_mount():
        self.config = ConfigManager.load_config()
        
        for source in self.config.sources:
            if source.enabled:
                self.aggregator.add_source(source)
        
        # Subscribe to new log entries
        self.log_channel = self.aggregator.subscribe()
        spawn_task(self.update_log_view())
    
    async function update_log_view():
        async for entry in self.log_channel:
            if self.filter_engine.apply(entry):
                self.log_view.append(format_entry(entry))
                self.status_bar.increment_count()
    
    function on_key(key: KeyEvent):
        match key:
            'q' -> self.quit()
            '/' -> self.show_search_dialog()
            'a' -> self.show_add_source_dialog()
            'f' -> self.show_filter_menu()
            'p' -> self.toggle_pause()
            ...
    
    function show_add_source_dialog():
        dialog = AddSourceDialog()
        result = await dialog.show()
        if result:
            ConfigManager.add_source(result)
            self.aggregator.add_source(result)
            self.sidebar.refresh()
}
```

### 6. LOG FORMATTERS / SYNTAX HIGHLIGHTING
```pseudo
class LogFormatter {
    rules: List<HighlightRule>
    
    struct HighlightRule {
        pattern: Regex
        style: TextStyle  # color, bold, etc.
    }
    
    DEFAULT_RULES = [
        { pattern: /\bERROR\b/i,    style: RED + BOLD },
        { pattern: /\bWARN(ING)?\b/i, style: YELLOW },
        { pattern: /\bINFO\b/i,     style: BLUE },
        { pattern: /\bDEBUG\b/i,    style: DIM },
        { pattern: /\bsudo\b/,      style: MAGENTA + BOLD },
        { pattern: /\bfailed\b/i,   style: RED },
        { pattern: /\bsegfault\b/i, style: RED + BOLD + BLINK },
        { pattern: /\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}/, style: CYAN },  # IPs
    ]
    
    function format(entry: LogEntry) -> StyledText:
        text = entry.message
        for rule in self.rules:
            text = apply_style(text, rule.pattern, rule.style)
        return text
}
```

## ═══════════════════════════════════════════════════════════════════
## DATA FLOW
## ═══════════════════════════════════════════════════════════════════

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  journalctl │     │  /var/log/  │     │  Custom     │
│  -o json -f │     │  auth.log   │     │  Paths      │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       ▼                   ▼                   ▼
┌──────────────────────────────────────────────────────┐
│              INGESTORS (async streams)               │
│  JournaldIngestor    FileIngestor    DirIngestor     │
└──────────────────────────┬───────────────────────────┘
                           │ LogEntry
                           ▼
┌──────────────────────────────────────────────────────┐
│                    AGGREGATOR                        │
│  - RingBuffer (keeps last N entries)                 │
│  - Broadcasts to all subscribers                     │
└──────────────────────────┬───────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────┐
│                  FILTER ENGINE                       │
│  - Level filter (WARN+)                              │
│  - Source filter (only nginx)                        │
│  - Regex search                                      │
└──────────────────────────┬───────────────────────────┘
                           │ (filtered)
                           ▼
┌──────────────────────────────────────────────────────┐
│                   LOG FORMATTER                      │
│  - Syntax highlighting                               │
│  - Timestamp formatting                              │
│  - Level badges                                      │
└──────────────────────────┬───────────────────────────┘
                           │ StyledText
                           ▼
┌──────────────────────────────────────────────────────┐
│                      TUI                             │
│  Sidebar │ LogView                   │ StatusBar     │
└──────────────────────────────────────────────────────┘
```

## ═══════════════════════════════════════════════════════════════════
## KEYBINDINGS (Draft)
## ═══════════════════════════════════════════════════════════════════

| Key | Action |
|-----|--------|
| `q` | Quit |
| `/` | Search (regex) |
| `Esc` | Clear search/filter |
| `a` | Add new log source |
| `d` | Delete selected source |
| `e` | Edit selected source |
| `f` | Filter menu |
| `1-5` | Quick filter by level (1=DEBUG, 5=CRITICAL) |
| `p` | Pause/Resume live feed |
| `g` | Go to top |
| `G` | Go to bottom (live tail) |
| `y` | Yank/copy selected line |
| `?` | Help |
| `Tab` | Switch focus (sidebar ↔ log view) |

## ═══════════════════════════════════════════════════════════════════
## CONFIG FILE STRUCTURE
## ═══════════════════════════════════════════════════════════════════

```yaml
# ~/.config/argus/config.yaml

general:
  theme: "dark"           # dark | light | custom
  max_buffer: 10000       # lines to keep in memory
  scroll_on_new: true
  timestamp_format: "%Y-%m-%d %H:%M:%S"

sources:
  - name: "System Journal"
    type: journald
    enabled: true
    filters:              # Optional journalctl filters
      - "-p warning"      # Priority warning+
    
  - name: "Auth Log"
    type: file
    path: "/var/log/auth.log"
    enabled: true
    
  - name: "Nginx Access"
    type: file
    path: "/var/log/nginx/access.log"
    enabled: true
    color: "#4fc3f7"
    
  - name: "Docker Containers"
    type: directory
    path: "/var/lib/docker/containers"
    glob: "*/*.log"
    enabled: false        # Disabled by default

highlight_rules:
  - pattern: "sudo"
    style: "bold magenta"
  - pattern: "segfault"
    style: "bold red blink"
```

## ═══════════════════════════════════════════════════════════════════
## OPEN QUESTIONS / TODO
## ═══════════════════════════════════════════════════════════════════

- [ ] Language choice: Python vs Rust vs Go
- [ ] How to handle log rotation mid-watch?
- [ ] Should we support remote log sources (SSH/syslog)?
- [ ] Export functionality? (save filtered logs to file)
- [ ] Bookmark/mark specific log lines?
- [ ] Session persistence? (remember scroll position)
