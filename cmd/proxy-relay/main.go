package main

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/anivaryam/proxy-relay/pkg/config"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	root := &cobra.Command{
		Use:     "proxy-relay",
		Short:   "Proxy client — route traffic through a remote SOCKS5/HTTP proxy",
		Version: version,
	}

	root.AddCommand(onCmd(), offCmd(), statusCmd(), configCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func onCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "on",
		Short: "Enable system proxy",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if cfg.ServerAddr == "" || cfg.AuthToken == "" {
				return fmt.Errorf("run 'proxy-relay config set-server <addr>' and 'proxy-relay config set-token <token>' first")
			}

			u, err := url.Parse(cfg.ServerAddr)
			if err != nil {
				return fmt.Errorf("invalid server address: %w", err)
			}

			host := u.Hostname()
			port := u.Port()
			if port == "" {
				if u.Scheme == "https" || u.Scheme == "socks5h" {
					port = "443"
				} else {
					port = "1080"
				}
			}

			socksAddr := fmt.Sprintf("socks5h://%s@%s", cfg.AuthToken, net.JoinHostPort(host, port))
			httpAddr := fmt.Sprintf("http://%s@%s", cfg.AuthToken, net.JoinHostPort(host, port))

			envContent := fmt.Sprintf(`export http_proxy="%s"
export https_proxy="%s"
export all_proxy="%s"
export HTTP_PROXY="%s"
export HTTPS_PROXY="%s"
export ALL_PROXY="%s"
export no_proxy="localhost,127.0.0.1,::1"
export NO_PROXY="localhost,127.0.0.1,::1"
`, httpAddr, httpAddr, socksAddr, httpAddr, httpAddr, socksAddr)

			envFile := envFilePath()
			if err := os.MkdirAll(filepath.Dir(envFile), 0700); err != nil {
				return err
			}
			if err := os.WriteFile(envFile, []byte(envContent), 0600); err != nil {
				return err
			}

			// Try gsettings for GUI apps
			if _, err := exec.LookPath("gsettings"); err == nil {
				exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "manual").Run()
				exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "host", host).Run()
				exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "port", port).Run()
			}

			fmt.Println("proxy enabled")
			fmt.Printf("  server: %s:%s\n", host, port)
			fmt.Println()
			fmt.Println("for terminal apps, run:")
			fmt.Printf("  source %s\n", envFile)
			return nil
		},
	}
}

func offCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "off",
		Short: "Disable system proxy",
		RunE: func(cmd *cobra.Command, args []string) error {
			os.Remove(envFilePath())

			if _, err := exec.LookPath("gsettings"); err == nil {
				exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "none").Run()
			}

			fmt.Println("proxy disabled")
			fmt.Println()
			fmt.Println("to clear terminal env vars, run:")
			fmt.Println("  unset http_proxy https_proxy all_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY no_proxy NO_PROXY")
			return nil
		},
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current proxy status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if cfg.ServerAddr == "" {
				fmt.Println("not configured (run 'proxy-relay config set-server <addr>')")
				return nil
			}

			fmt.Printf("server: %s\n", cfg.ServerAddr)

			if _, err := os.Stat(envFilePath()); err == nil {
				fmt.Println("status: ON (env file exists)")
			} else {
				fmt.Println("status: OFF")
			}

			// Check shell env
			if v := os.Getenv("all_proxy"); v != "" {
				fmt.Printf("shell:  all_proxy=%s\n", v)
			}

			return nil
		},
	}
}

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage proxy-relay configuration",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "set-server <addr>",
		Short: "Set proxy server address (e.g. socks5h://host:port)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			addr := args[0]
			if !strings.Contains(addr, "://") {
				addr = "socks5h://" + addr
			}
			cfg.ServerAddr = addr
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Printf("server set to %s\n", cfg.ServerAddr)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "set-token <token>",
		Short: "Set auth token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.AuthToken = args[0]
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Println("token saved")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			fmt.Printf("server_addr: %s\n", cfg.ServerAddr)
			if cfg.AuthToken != "" {
				fmt.Printf("auth_token:  %s...%s\n", cfg.AuthToken[:3], cfg.AuthToken[len(cfg.AuthToken)-3:])
			} else {
				fmt.Println("auth_token:  (not set)")
			}
			return nil
		},
	})

	return cmd
}

func envFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".proxy-relay", "proxy.env")
}
