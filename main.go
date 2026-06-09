package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/Rulunchic/ObGoData/internal/analyze"
	"github.com/Rulunchic/ObGoData/internal/render"
	"github.com/Rulunchic/ObGoData/internal/scanner"
)

func main() {
	var (
		vaultPath = flag.String("vault", ".", "path to Obsidian vault")
		port      = flag.Int("port", 0, "HTTP server port (0 = random)")
		noOpen    = flag.Bool("no-open", false, "don't open browser")
		verbose   = flag.Bool("v", false, "verbose logging")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	if *verbose {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}

	if err := run(logger, *vaultPath, *port, *noOpen); err != nil {
		logger.Error("failed", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger, vaultPath string, port int, noOpen bool) error {
	start := time.Now()

	logger.Info("scanning vault", "path", vaultPath)
	result, err := scanner.Scan(vaultPath)
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}
	logger.Info("scan complete",
		"notes", len(result.Notes),
		"dirs", len(result.Dirs),
		"elapsed", time.Since(start).Round(time.Millisecond),
	)

	scanTime := time.Since(start)

	logger.Info("analyzing")
	stats := analyze.Compute(result)
	logger.Info("analysis complete",
		"words", stats.TotalWords,
		"links", stats.TotalLinks,
		"attachments", stats.TotalAttachs,
	)

	logger.Info("generating site")
	outDir, err := render.Generate(stats, vaultPath)
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}
	logger.Info("site generated", "dir", outDir)

	totalTime := time.Since(start)

	fmt.Println()
	fmt.Printf("  Notes:        %d\n", stats.TotalNotes)
	fmt.Printf("  Words:        %d\n", stats.TotalWords)
	fmt.Printf("  Links:        %d\n", stats.TotalLinks)
	fmt.Printf("  Attachments:  %d\n", stats.TotalAttachs)
	fmt.Printf("  Size:         %s\n", formatSize(stats.TotalSize))
	fmt.Printf("  Dirs:         %d\n", stats.DirCount)
	fmt.Printf("  Scan:         %s\n", scanTime.Round(time.Millisecond))
	fmt.Printf("  Total:        %s\n", totalTime.Round(time.Millisecond))
	fmt.Println()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	actualAddr := listener.Addr().String()
	fmt.Printf("  Serving:      http://%s\n", actualAddr)
	fmt.Println()

	if !noOpen {
		openBrowser("http://" + actualAddr)
	}

	logger.Info("serving", "addr", actualAddr)
	return http.Serve(listener, http.FileServer(http.Dir(outDir)))
}

func openBrowser(url string) {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	_ = cmd.Start()
}

func formatSize(n int64) string {
	if n >= 1073741824 {
		return fmt.Sprintf("%.1f GB", float64(n)/1073741824)
	}
	if n >= 1048576 {
		return fmt.Sprintf("%.1f MB", float64(n)/1048576)
	}
	if n >= 1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	return fmt.Sprintf("%d B", n)
}
