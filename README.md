# headless_browser

A simple headless browser library powered by go-rod, with built-in stealth mode support.

> **This is a hard fork of [xpzouying/headless_browser](https://github.com/xpzouying/headless_browser)** (forked at v0.4.0).
> The module path is `github.com/alex-soldatkin/headless_browser`, so a consumer that already
> imports the upstream path only needs a `replace` directive and no source changes:
>
> ```
> require github.com/xpzouying/headless_browser v0.4.0
> replace github.com/xpzouying/headless_browser => github.com/alex-soldatkin/headless_browser v0.5.0
> ```
>
> The upstream API is kept backward compatible.

## Installation

```bash
go get github.com/alex-soldatkin/headless_browser
```

## Usage

```go
package main

import (
    "time"
    
    "github.com/alex-soldatkin/headless_browser"
)

func main() {
    // Create browser with default settings (headless mode)
    browser := headless_browser.New()
    defer browser.Close()
    
    // Create a new page
    page := browser.NewPage()
    defer page.Close()
    
    // Navigate to a website
    page.Timeout(30 * time.Second).
        MustNavigate("https://example.com").
        MustWaitStable()
}
```

## Configuration Options

```go
// Run in non-headless mode (visible browser)
browser := headless_browser.New(
    headless_browser.WithHeadless(false),
)

// Set custom user agent
browser := headless_browser.New(
    headless_browser.WithUserAgent("Custom User Agent"),
)

// Set cookies (JSON format)
browser := headless_browser.New(
    headless_browser.WithCookies(`[{"name":"session","value":"abc123","domain":"example.com"}]`),
)

// Combine multiple options
browser := headless_browser.New(
    headless_browser.WithHeadless(false),
    headless_browser.WithUserAgent("Custom User Agent"),
    headless_browser.WithCookies(cookiesJSON),
)
```

## Fork additions

### Persistent profiles: `WithUserDataDir`

Upstream's `Close()` calls `launcher.Cleanup()`, which does an unconditional
`os.RemoveAll` of the user data directory — a persistent Chrome profile would be
destroyed on every clean shutdown. (rod's `rod-keep-user-data-dir` flag does not
help: it is only honoured by the remote launcher `Manager`.)

`WithUserDataDir` sets the flag *and* marks the directory as owned by the
caller, so `Close()` leaves it alone while still waiting for the browser process
to exit:

```go
browser := headless_browser.New(
    headless_browser.WithUserDataDir("/var/lib/myapp/chrome-profile"),
)
defer browser.Close() // profile survives
```

Without the option, behaviour is unchanged: rod's temp profile is still removed.

### Escape hatch: `WithLauncherHook`

Runs against the rod launcher after every other flag is applied and immediately
before `MustLaunch`, so you can reach anything rod exposes without a bespoke
option per need:

```go
headless_browser.WithLauncherHook(func(l *launcher.Launcher) {
    l.Delete("enable-automation")
    l.Env("LANG=zh_CN.UTF-8")
    l.Preferences(`{"profile":{"exit_type":"Normal"}}`)
})
```

### Per-page setup: `WithPageHook`

Runs on every page returned by `NewPage`, after the consistent UA override. An
error is logged and does not abort page creation.

```go
headless_browser.WithPageHook(func(p *rod.Page) error {
    return proto.EmulationSetDeviceMetricsOverride{
        Width: 1280, Height: 800, DeviceScaleFactor: 1,
    }.Call(p)
})
```

Multiple `WithLauncherHook` / `WithPageHook` options compose, in the order given.

### `Browser.Rod()`

Returns the underlying `*rod.Browser` for browser-level CDP calls this package
does not wrap.

## Testing

The browser tests need a real Chrome/Chromium. They use `launcher.LookPath()`
and are skipped when nothing is found; point them at a specific binary with:

```bash
HEADLESS_BROWSER_BIN=/path/to/Chromium go test ./...
```

## Example

```go
package main

import (
    "time"
    
    "github.com/alex-soldatkin/headless_browser"
)

func main() {
    // Create browser instance
    browser := headless_browser.New(headless_browser.WithHeadless(false))
    defer browser.Close()

    // Create new page with stealth mode
    page := browser.NewPage()
    defer page.Close()

    // Navigate and wait for page to be stable
    page.Timeout(30 * time.Second).
        MustNavigate("https://www.haha.ai").
        MustWaitStable()

    // Additional page operations can be performed here
    time.Sleep(1 * time.Second)
}
```
