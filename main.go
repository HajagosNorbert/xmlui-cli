package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"xmlui-mcp/pkg/xmlui"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "xmlui",
	Short: "xmlui is a CLI for XMLUI components",
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Starts the model context protocol server",
	Run: func(cmd *cobra.Command, args []string) {
		// Get the current directory to find XMLUI source
		currentDir, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get current directory: %v", err)
		}

		// Determine XMLUI directory
		xmluiDir := mcpXMLUIDir
		if xmluiDir == "" {
			// Try default locations
			xmluiDir = filepath.Join(currentDir, "..", "xmlui")
			if _, err := os.Stat(xmluiDir); os.IsNotExist(err) {
				log.Fatalf("XMLUI directory not found at %s, trying alternative locations", xmluiDir)
			}
		}

		// Configure the XMLUI MCP server
		config := xmlui.ServerConfig{
			XMLUIDir:      xmluiDir,
			HTTPMode:      mcpHTTPMode,
			Port:          mcpPort,
			AnalyticsFile: mcpAnalyticsFile,
		}

		// Create the server instance using the local library
		server, err := xmlui.NewServer(config)
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
			fmt.Println("This template provides a complete business application for invoice management.")
		case "hello-world":
			fmt.Println("Scaffolding hello-world project...")
			fmt.Println("This template provides a minimal app to get you started with XMLUI.")
		default:
			fmt.Printf("Error: unknown template %q\n", template)
			fmt.Println("Available templates:")
			fmt.Println("  hello-world    - A minimal app to get you started with XMLUI")
			fmt.Println("  xmlui-invoice  - A complete business application for invoice management")
			os.Exit(1)
		}
	},
}

var (
	mcpXMLUIDir      string
	mcpPort          string
	mcpAnalyticsFile string
	mcpHTTPMode      bool
)

func init() {
	// MCP command flags
	mcpCmd.Flags().StringVar(&mcpXMLUIDir, "xmlui-dir", "", "Path to XMLUI source directory")
	mcpCmd.Flags().StringVar(&mcpPort, "port", "9090", "Port to run HTTP server on")
	mcpCmd.Flags().StringVar(&mcpAnalyticsFile, "analytics-file", "./my-app-analytics.json", "Path to analytics file")
	mcpCmd.Flags().BoolVar(&mcpHTTPMode, "http", false, "Run as HTTP server")

	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(scaffoldCmd)
}
