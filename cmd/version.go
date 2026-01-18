package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ASRagab/agentpipe/internal/version"
	"github.com/ASRagab/agentpipe/pkg/adapters"
	_ "github.com/ASRagab/agentpipe/pkg/adapters/api"
	_ "github.com/ASRagab/agentpipe/pkg/adapters/cli"
)

var (
	checkUpdate  bool
	showAdapters bool
	showV2Info   bool
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long: `Display the current version of agentpipe, v2 engine version, and check for updates.

Examples:
  agentpipe version              # Show basic version info
  agentpipe version --adapters   # Show registered adapters and capabilities
  agentpipe version --v2         # Show v2 engine details
  agentpipe version --all        # Show all version information`,
	Run: runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().BoolVar(&checkUpdate, "check-update", true, "Check for newer versions")
	versionCmd.Flags().BoolVar(&showAdapters, "adapters", false, "Show registered adapters")
	versionCmd.Flags().BoolVar(&showV2Info, "v2", false, "Show v2 engine details")
	versionCmd.Flags().Bool("all", false, "Show all version information")
}

func runVersion(cmd *cobra.Command, args []string) {
	showAll, _ := cmd.Flags().GetBool("all")

	// Main version info
	fmt.Println(version.GetVersionString())

	// Show running mode indicator
	printRunningModeIndicator()

	// Show v2 engine info
	if showV2Info || showAll {
		printV2EngineInfo()
	}

	// Show adapter information
	if showAdapters || showAll {
		printAdapterInfo()
	}

	// Check for updates
	if checkUpdate {
		fmt.Println("\n🔍 Checking for updates...")
		hasUpdate, latestVersion, err := version.CheckForUpdate()

		if err != nil {
			// Only show error if it's not a silent failure
			if err.Error() != "" {
				fmt.Printf("   Could not check for updates: %v\n", err)
			}
			return
		}

		if hasUpdate {
			fmt.Printf("\n📦 Update available!\n")
			fmt.Printf("   Current version: %s (out of date)\n", version.GetShortVersion())
			fmt.Printf("   Latest version:  %s\n", latestVersion)
			fmt.Printf("\n   Update with: brew upgrade agentpipe\n")
			fmt.Printf("   Or download from: https://github.com/ASRagab/agentpipe/releases/latest\n")
		} else if latestVersion != "" {
			fmt.Printf("   You're running the latest version! (%s)\n", latestVersion)
		} else {
			// Couldn't determine the latest version
			fmt.Printf("   Update check unavailable at this time\n")
		}
	}
}

// printRunningModeIndicator shows whether v1 or v2 mode is the default.
func printRunningModeIndicator() {
	v2Default := os.Getenv("AGENTPIPE_V2") == "true"
	if v2Default {
		fmt.Println("Default mode: v2 (parallel execution engine)")
	} else {
		fmt.Println("Default mode: v1 (use --v2 flag for parallel execution)")
	}
}

// printV2EngineInfo displays v2 engine version and features.
func printV2EngineInfo() {
	v2Version, features := version.GetV2EngineInfo()

	fmt.Printf("\nv2 Engine Version: %s\n", v2Version)
	fmt.Println(strings.Repeat("-", 40))
	fmt.Println("Features:")
	for _, feature := range features {
		fmt.Printf("  - %s\n", feature)
	}
}

// printAdapterInfo displays registered adapters and their capabilities.
func printAdapterInfo() {
	registeredAdapters := adapters.List()

	fmt.Printf("\nRegistered Adapters (%d):\n", len(registeredAdapters))
	fmt.Println(strings.Repeat("-", 40))

	if len(registeredAdapters) == 0 {
		fmt.Println("  No adapters registered")
		return
	}

	for _, name := range registeredAdapters {
		adapter, err := adapters.Get(name)
		if err != nil {
			fmt.Printf("  %s: error loading (%v)\n", name, err)
			continue
		}

		availableIcon := "x"
		if adapter.IsAvailable() {
			availableIcon = "+"
		}

		adapterType := getAdapterType(name)
		fmt.Printf("  [%s] %s (%s)\n", availableIcon, name, adapterType)
	}

	fmt.Println("\n  Legend: [+] available  [x] not available")
}

// getAdapterType determines the adapter type based on name conventions.
func getAdapterType(name string) string {
	if strings.HasSuffix(name, "-cli") {
		return "CLI"
	}
	if strings.HasSuffix(name, "-api") {
		return "API"
	}
	// Check for known API adapters
	apiAdapters := []string{"openrouter", "claude-api"}
	for _, api := range apiAdapters {
		if name == api {
			return "API"
		}
	}
	return "unknown"
}
