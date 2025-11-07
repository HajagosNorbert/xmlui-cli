package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"xmlui-mcp/pkg/xmluimcp"
	"xmlui-test-server"
)

func launchBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}

	err := exec.Command(cmd, args...).Start()
	if err != nil {
		log.Printf("Failed to launch browser: %v", err)
	}
}

func injectPgPort(pgConnStr, pgPort string) string {
	if pgConnStr == "" || pgPort == "" {
		return pgConnStr
	}
	if strings.HasPrefix(pgConnStr, "postgres://") || strings.HasPrefix(pgConnStr, "postgresql://") {
		u, err := url.Parse(pgConnStr)
		if err == nil {
			if u.Port() == "" || u.Port() != pgPort {
				u.Host = u.Hostname() + ":" + pgPort
				return u.String()
			}
		}
		return pgConnStr
	}
	re := regexp.MustCompile(`port=\d+`)
	if re.MatchString(pgConnStr) {
		return re.ReplaceAllString(pgConnStr, "port="+pgPort)
	}
	return pgConnStr + " port=" + pgPort
}

func downloadAndExtractZip(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	r, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return err
	}

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "xmlui",
	Short: "An all-in-one tool for working with xmlui.",
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Starts the model context protocol server",
	Run: func(cmd *cobra.Command, args []string) {

		// Configure the XMLUI MCP server
		config := xmluimcp.ServerConfig{
			XMLUIDir:      mcpXMLUIDir,
			HTTPMode:      mcpHTTPMode,
			Port:          mcpPort,
			AnalyticsFile: mcpAnalyticsFile,
		}

		// Create the server instance using the local library
		server, err := xmluimcp.NewServer(config)
		if err != nil {
			log.Fatalf("Failed to create XMLUI MCP server: %v", err)
		}

		server.PrintStartupInfo()

		if mcpHTTPMode {
			if err := server.ServeHTTP(); err != nil {
				log.Fatalf("Server error: %v", err)
			}
		} else {
			if err := server.ServeStdio(); err != nil {
				log.Fatalf("Stdio server error: %v", err)
			}
		}
	},
}

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold [template]",
	Short: "Scaffolds a new project from a template",
	Long: `Scaffolds a new project from available templates.

Available templates:
  hello-world    - A minimal app to get you started with XMLUI
  xmlui-invoice  - A complete business application for invoice management`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		template := args[0]
		switch template {
		case "xmlui-invoice":
			fmt.Println("Scaffolding xmlui-invoice project...")

			zipURL := "https://github.com/xmlui-org/xmlui-invoice/archive/refs/heads/hajagosnorbert/demo.zip"
			targetDir := "xmlui-invoice"

			if _, err := os.Stat(targetDir); err == nil {
				log.Fatalf("Directory %s already exists", targetDir)
			}

			fmt.Println("Downloading and extracting xmlui-invoice...")
			if err := downloadAndExtractZip(zipURL, "."); err != nil {
				log.Fatalf("Failed to download and extract: %v", err)
			}

			extractedDir := "xmlui-invoice-hajagosnorbert-demo"
			if err := os.Rename(extractedDir, targetDir); err != nil {
				log.Fatalf("Failed to rename directory: %v", err)
			}

			fmt.Println("\nScaffolding complete!")
			fmt.Printf("Project created in: %s\n", targetDir)
			fmt.Println("\nNavigate and start the project by running:\n")
			fmt.Printf("  cd %s && xmlui serve\n", targetDir)

		case "hello-world":
			fmt.Println("Scaffolding hello-world not yet implemented")
		default:
			fmt.Printf("Error: unknown template %q\n", template)
			fmt.Println("Available templates:")
			fmt.Println("  hello-world    - A minimal app to get you started with XMLUI")
			fmt.Println("  xmlui-invoice  - A complete business application for invoice management")
			os.Exit(1)
		}
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve [client-dir]",
	Short: "Starts the XMLUI server",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log.SetFlags(log.Lshortfile | log.LstdFlags)
		log.Println("Server starting...")

		pwd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Working directory: %s", pwd)

		clientDir := "client"
		if len(args) > 0 {
			clientDir = args[0]
		}

		showResponsesEnabled := serveShowResponses
		finalPgConnStr := injectPgPort(servePgConnStr, servePgPort)

		config := xmluibackend.ServerConfig{
			DBPath:        serveDBPath,
			PgConnStr:     finalPgConnStr,
			ExtensionPath: serveExtension,
			APIDescPath:   serveAPIDesc,
			ShowResponses: showResponsesEnabled,
		}

		server, err := xmluibackend.NewServer(config)
		if err != nil {
			log.Fatal(err)
		}
		defer server.Close()

		mux := http.NewServeMux()
		server.SetupRoutes(mux, clientDir)

		log.Printf("Server configuration:")
		log.Printf("- Port: %s", servePort)
		log.Printf("- API Description: %s", serveAPIDesc)
		log.Printf("- Extension: %s", serveExtension)
		log.Printf("- Show Responses: %v", showResponsesEnabled)
		log.Printf("- Client Directory: %s", clientDir)
		if servePgConnStr != "" {
			log.Printf("- Database: PostgreSQL")
		} else {
			os.Setenv("STEAMPIPE_CACHE", "false")
			log.Printf("- Database: SQLite (%s)", serveDBPath)
		}

		log.Printf("Opening web browser...")
		launchBrowser(fmt.Sprintf("http://localhost:%s", servePort))

		log.Printf("Server listening on http://localhost:%s...", servePort)
		if err := http.ListenAndServe("127.0.0.1:"+servePort, xmluibackend.CORSMiddleware(mux)); err != nil {
			log.Fatal(err)
		}
	},
}

var (
	mcpXMLUIDir      string
	mcpPort          string
	mcpAnalyticsFile string
	mcpHTTPMode      bool

	servePort         string
	serveExtension    string
	serveAPIDesc      string
	serveDBPath       string
	serveShowResponses bool
	servePgConnStr    string
	servePgPort       string
)

func init() {
	// MCP command flags
	mcpCmd.Flags().StringVar(&mcpXMLUIDir, "xmlui-dir", "", "Path to XMLUI source directory")
	mcpCmd.Flags().StringVar(&mcpPort, "port", "9090", "Port to run HTTP server on")
	mcpCmd.Flags().StringVar(&mcpAnalyticsFile, "analytics-file", "./mcp-analytics.json", "Path to analytics file")
	mcpCmd.Flags().BoolVar(&mcpHTTPMode, "http", false, "Run as HTTP server")

	serveCmd.Flags().StringVar(&servePort, "port", "8080", "Port to run the server on")
	serveCmd.Flags().StringVarP(&servePort, "p", "p", "8080", "Port to run the server on (shorthand)")
	serveCmd.Flags().StringVar(&serveExtension, "extension", "", "Path to SQLite extension to load")
	serveCmd.Flags().StringVar(&serveAPIDesc, "api", "api.json", "Path to API description file")
	serveCmd.Flags().StringVar(&serveDBPath, "db", "data.db", "Path to SQLite database file")
	serveCmd.Flags().BoolVar(&serveShowResponses, "show-responses", false, "Enable logging of SQL query responses")
	serveCmd.Flags().BoolVarP(&serveShowResponses, "s", "s", false, "Enable logging of SQL query responses (shorthand)")
	serveCmd.Flags().StringVar(&servePgConnStr, "pg-conn", "", "PostgreSQL connection string")
	serveCmd.Flags().StringVar(&servePgPort, "pg-port", "", "PostgreSQL port (overrides port in --pg-conn)")

	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(scaffoldCmd)
	rootCmd.AddCommand(serveCmd)
}
