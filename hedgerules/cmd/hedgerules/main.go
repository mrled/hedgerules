package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore"
	"github.com/micahrl/hedgerules/internal/functions"
	"github.com/micahrl/hedgerules/internal/hugo"
	"github.com/micahrl/hedgerules/internal/kvs"
)

var version = "dev"

type config struct {
	OutputDir string `toml:"output-dir"`
	Region    string `toml:"region"`

	Redirects kvsConfig       `toml:"redirects"`
	Headers   kvsConfig       `toml:"headers"`
	Functions functionsConfig `toml:"functions"`
}

type kvsConfig struct {
	KVSName string `toml:"kvs-name"`
}

type functionsConfig struct {
	RequestName  string `toml:"request-name"`
	ResponseName string `toml:"response-name"`
	DebugHeaders bool   `toml:"debug-headers"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "deploy":
		runDeploy(os.Args[2:])
	case "version":
		fmt.Println(version)
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: hedgerules <command> [flags]\n\nCommands:\n  deploy   Sync KVS data and deploy CloudFront Functions\n  version  Print version\n\nRun 'hedgerules deploy --help' for deploy flags.\n")
}

func runDeploy(args []string) {
	fs := flag.NewFlagSet("deploy", flag.ExitOnError)
	configPath := fs.String("config", "hedgerules.toml", "path to config file")
	outputDir := fs.String("output-dir", "", "Hugo build output directory")
	redirectsKVS := fs.String("redirects-kvs-name", "", "CloudFront KVS name for redirects")
	headersKVS := fs.String("headers-kvs-name", "", "CloudFront KVS name for headers")
	requestFunc := fs.String("request-function-name", "", "CloudFront Function name for viewer-request")
	responseFunc := fs.String("response-function-name", "", "CloudFront Function name for viewer-response")
	dryRun := fs.Bool("dry-run", false, "parse and validate only, print plan")
	region := fs.String("region", "", "AWS region override")
	debugHeaders := fs.Bool("debug-headers", false, "inject debug headers into viewer-response function")
	fs.Parse(args)

	// Load config file
	cfg := loadConfig(*configPath)

	// CLI flags override config
	if *outputDir != "" {
		cfg.OutputDir = *outputDir
	}
	if *redirectsKVS != "" {
		cfg.Redirects.KVSName = *redirectsKVS
	}
	if *headersKVS != "" {
		cfg.Headers.KVSName = *headersKVS
	}
	if *requestFunc != "" {
		cfg.Functions.RequestName = *requestFunc
	}
	if *responseFunc != "" {
		cfg.Functions.ResponseName = *responseFunc
	}
	if *region != "" {
		cfg.Region = *region
	}
	if *debugHeaders {
		cfg.Functions.DebugHeaders = true
	}

	// Validate required config
	if cfg.OutputDir == "" {
		fatal("output-dir is required (set in config file or via --output-dir)")
	}
	if !*dryRun {
		if cfg.Redirects.KVSName == "" {
			fatal("redirects-kvs-name is required (set in config file or via --redirects-kvs-name)")
		}
		if cfg.Headers.KVSName == "" {
			fatal("headers-kvs-name is required (set in config file or via --headers-kvs-name)")
		}
		if cfg.Functions.RequestName == "" {
			fatal("request-function-name is required (set in config file or via --request-function-name)")
		}
		if cfg.Functions.ResponseName == "" {
			fatal("response-function-name is required (set in config file or via --response-function-name)")
		}
	}

	// Step 1: Parse Hugo output
	fmt.Fprintf(os.Stderr, "Scanning directories in %s...\n", cfg.OutputDir)
	dirEntries, err := hugo.ScanDirectories(cfg.OutputDir)
	if err != nil {
		fatal("scanning directories: %v", err)
	}
	fmt.Fprintf(os.Stderr, "Found %d directory redirects\n", len(dirEntries))

	fmt.Fprintf(os.Stderr, "Parsing _hedge_redirects.txt...\n")
	fileEntries, err := hugo.ParseRedirects(cfg.OutputDir)
	if err != nil {
		fatal("parsing _hedge_redirects.txt: %v", err)
	}
	fmt.Fprintf(os.Stderr, "Found %d file redirects\n", len(fileEntries))

	mergedEntries := hugo.MergeRedirects(dirEntries, fileEntries)
	fmt.Fprintf(os.Stderr, "Total redirects after merge: %d\n", len(mergedEntries))

	fmt.Fprintf(os.Stderr, "Resolving redirect chains...\n")
	redirectEntries, err := hugo.ResolveChains(mergedEntries)
	if err != nil {
		fatal("resolving redirect chains: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Parsing _hedge_headers.json...\n")
	headerEntries, err := hugo.ParseHeaders(cfg.OutputDir)
	if err != nil {
		fatal("parsing _hedge_headers.json: %v", err)
	}
	fmt.Fprintf(os.Stderr, "Found %d header entries\n", len(headerEntries))

	// Step 2: Validate
	redirectData := &kvs.Data{Entries: redirectEntries}
	headerData := &kvs.Data{Entries: headerEntries}

	var validationErrors []kvs.ValidationError
	validationErrors = append(validationErrors, redirectData.Validate()...)
	validationErrors = append(validationErrors, headerData.Validate()...)

	if len(validationErrors) > 0 {
		fmt.Fprintf(os.Stderr, "\nValidation errors:\n")
		for _, e := range validationErrors {
			fmt.Fprintf(os.Stderr, "  %s: %s\n", e.Key, e.Message)
		}
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Validation passed\n")

	// Report KVS capacity usage
	redirectStats := redirectData.Stats()
	headerStats := headerData.Stats()
	fmt.Fprintf(os.Stderr, "\nKVS capacity:\n")
	fmt.Fprintf(os.Stderr, "  Redirects: %d keys, %d / %d bytes (%.1f%%)\n",
		redirectStats.NumKeys, redirectStats.TotalBytes, kvs.MaxTotalBytes,
		float64(redirectStats.TotalBytes)/float64(kvs.MaxTotalBytes)*100)
	fmt.Fprintf(os.Stderr, "  Headers:   %d keys, %d / %d bytes (%.1f%%)\n",
		headerStats.NumKeys, headerStats.TotalBytes, kvs.MaxTotalBytes,
		float64(headerStats.TotalBytes)/float64(kvs.MaxTotalBytes)*100)

	// Step 3: Dry run - print plan and exit
	if *dryRun {
		fmt.Println("\n=== Redirects ===")
		for _, e := range redirectEntries {
			fmt.Printf("%s -> %s\n", e.Key, e.Value)
		}
		fmt.Println("\n=== Headers ===")
		for _, e := range headerEntries {
			fmt.Printf("%s:\n%s\n---\n", e.Key, e.Value)
		}
		fmt.Fprintf(os.Stderr, "\nDry run complete. No changes made.\n")
		return
	}

	// Step 4: Set up AWS clients
	ctx := context.Background()
	var awsOpts []func(*awsconfig.LoadOptions) error
	if cfg.Region != "" {
		awsOpts = append(awsOpts, awsconfig.WithRegion(cfg.Region))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsOpts...)
	if err != nil {
		fatal("loading AWS config: %v", err)
	}

	cfClient := cloudfront.NewFromConfig(awsCfg)
	kvsClient := cloudfrontkeyvaluestore.NewFromConfig(awsCfg)

	// Step 5: Resolve KVS ARNs
	fmt.Fprintf(os.Stderr, "Resolving KVS ARNs...\n")
	redirectsARN, err := functions.ResolveKVSARN(ctx, cfClient, cfg.Redirects.KVSName)
	if err != nil {
		fatal("resolving redirects KVS: %v", err)
	}
	fmt.Fprintf(os.Stderr, "Redirects KVS: %s\n", redirectsARN)

	headersARN, err := functions.ResolveKVSARN(ctx, cfClient, cfg.Headers.KVSName)
	if err != nil {
		fatal("resolving headers KVS: %v", err)
	}
	fmt.Fprintf(os.Stderr, "Headers KVS: %s\n", headersARN)

	// Step 6: Sync redirects KVS
	fmt.Fprintf(os.Stderr, "Syncing redirects KVS...\n")
	existingRedirects, redirectEtag, err := kvs.FetchExistingKeys(ctx, kvsClient, redirectsARN)
	if err != nil {
		fatal("fetching existing redirects: %v", err)
	}
	redirectPlan := kvs.ComputeSyncPlan(redirectData, existingRedirects)
	fmt.Fprintf(os.Stderr, "Redirects: %d puts, %d deletes\n", len(redirectPlan.Puts), len(redirectPlan.Deletes))
	if err := kvs.Sync(ctx, kvsClient, redirectsARN, redirectEtag, redirectPlan); err != nil {
		fatal("syncing redirects: %v", err)
	}

	// Step 7: Sync headers KVS
	fmt.Fprintf(os.Stderr, "Syncing headers KVS...\n")
	existingHeaders, headerEtag, err := kvs.FetchExistingKeys(ctx, kvsClient, headersARN)
	if err != nil {
		fatal("fetching existing headers: %v", err)
	}
	headerPlan := kvs.ComputeSyncPlan(headerData, existingHeaders)
	fmt.Fprintf(os.Stderr, "Headers: %d puts, %d deletes\n", len(headerPlan.Puts), len(headerPlan.Deletes))
	if err := kvs.Sync(ctx, kvsClient, headersARN, headerEtag, headerPlan); err != nil {
		fatal("syncing headers: %v", err)
	}

	// Step 8: Deploy CloudFront Functions
	fmt.Fprintf(os.Stderr, "Deploying viewer-request function...\n")
	requestCode := functions.BuildFunctionCode(functions.ViewerRequestJS, redirectsARN, false)
	if err := functions.DeployFunction(ctx, cfClient, cfg.Functions.RequestName, requestCode, redirectsARN); err != nil {
		fatal("deploying viewer-request function: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Deploying viewer-response function...\n")
	responseCode := functions.BuildFunctionCode(functions.ViewerResponseJS, headersARN, cfg.Functions.DebugHeaders)
	if err := functions.DeployFunction(ctx, cfClient, cfg.Functions.ResponseName, responseCode, headersARN); err != nil {
		fatal("deploying viewer-response function: %v", err)
	}

	fmt.Fprintf(os.Stderr, "\nDeploy complete.\n")
}

func loadConfig(path string) config {
	var cfg config
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return cfg // No config file, use defaults/flags
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		fatal("reading config file %s: %v", path, err)
	}
	return cfg
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
