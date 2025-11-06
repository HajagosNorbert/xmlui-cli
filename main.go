package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
		fmt.Println("Starting model context protocol server...")
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

func init() {
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(scaffoldCmd)
}
