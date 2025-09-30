package main

import (
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
    "time"
)

func copyFile(src, dst string) error {
    in, err := os.Open(src)
    if err != nil {
        return err
    }
    defer in.Close()
    out, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer out.Close()
    _, err = io.Copy(out, in)
    if err != nil {
        return err
    }
    return out.Sync()
}

func fileModTime(path string) time.Time {
    fi, err := os.Stat(path)
    if err != nil {
        return time.Time{}
    }
    return fi.ModTime()
}

func main() {
    args := os.Args[1:]
    logfile := "C:\\mstsc_proxy.log"
    f, _ := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if f != nil {
        defer f.Close()
    }

    ts := time.Now().Format("2006-01-02 15:04:05")
    if f != nil {
        fmt.Fprintf(f, "\n[%s] called with args: %v\n", ts, args)
    }

    outDir := `C:\mstsc_hook`
    _ = os.MkdirAll(outDir, 0755)

    var srcRdp string
    for _, a := range args {
        aTrim := strings.Trim(a, `"`)
        if strings.HasSuffix(strings.ToLower(aTrim), ".rdp") {
            srcRdp = aTrim
            break
        }
    }

    if srcRdp == "" {
        if f != nil {
            fmt.Fprintln(f, "no .rdp arg found; exiting")
        }
        return
    }

    base := filepath.Base(srcRdp)
    dstHook := filepath.Join(outDir, base)
    dstRoot := filepath.Join(`C:\`, base)

    if err := copyFile(srcRdp, dstHook); err != nil {
        if f != nil {
            fmt.Fprintf(f, "initial copy to hook failed: %v\n", err)
        }
    } else if f != nil {
        fmt.Fprintf(f, "copied to hook: %s\n", dstHook)
    }

    if err := copyFile(srcRdp, dstRoot); err == nil {
        if f != nil {
            fmt.Fprintf(f, "copied to root C: %s\n", dstRoot)
        }
    }

    lastMod := fileModTime(srcRdp)

    alivePath := filepath.Join(outDir, "alive")
    sessionEndPath := filepath.Join(outDir, "session_end")

    maxWait := 8 * time.Hour
    start := time.Now()

    for {
        now := time.Now().Format("2006-01-02 15:04:05")
        if f != nil {
            fmt.Fprintf(f, "[%s] heartbeat\n", now)
        }
        _ = os.WriteFile(alivePath, []byte(now+"\n"), 0644)

        curMod := fileModTime(srcRdp)
        if !curMod.IsZero() && curMod.After(lastMod) {
            if err := copyFile(srcRdp, dstHook); err != nil {
                if f != nil {
                    fmt.Fprintf(f, "recopy to hook failed: %v\n", err)
                }
            } else if f != nil {
                fmt.Fprintf(f, "re-copied to hook at %s\n", time.Now().Format(time.RFC3339))
            }
            if err := copyFile(srcRdp, dstRoot); err == nil {
                if f != nil {
                    fmt.Fprintf(f, "re-copied to C: root\n")
                }
            }
            lastMod = curMod
        }

        if _, err := os.Stat(sessionEndPath); err == nil {
            if f != nil {
                fmt.Fprintln(f, "session_end found -> exiting")
            }
            break
        }

        if time.Since(start) > maxWait {
            if f != nil {
                fmt.Fprintln(f, "max wait exceeded -> exiting")
            }
            break
        }

        time.Sleep(1 * time.Second)
    }

    if f != nil {
        fmt.Fprintln(f, "mstsc_stub exiting")
    }
}