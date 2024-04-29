/*
Copyright © 2022 codeperf.io <hello@codeperf.io>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"os"

	"github.com/spf13/viper"
)

var cfgFile string
var bench string
var gitOrg string
var gitRepo string
var gitBranch string
var gitCommit string
var localFilename string
var codeperfUrl string
var codeperfApiUrl string
var benchTime string
var local bool
var longDescription = `                  __                     ____        _
  _________  ____/ /__  ____  ___  _____/ __/       (_)___
 / ___/ __ \/ __  / _ \/ __ \/ _ \/ ___/ /_        / / __ \
/ /__/ /_/ / /_/ /  __/ /_/ /  __/ /  / __/  _    / / /_/ /
\___/\____/\__,_/\___/ .___/\___/_/  /_/    (_)  /_/\____/
                    /_/

Export and persist Go's performance data into https://codeperf.io.`

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "codeperf",
	Short: "Export and persist Go's performance into https://codeperf.io.",
	Long:  longDescription,
}

var cmdPrint = &cobra.Command{
	Use:   "test",
	Short: "Print anything to the screen",
	Long: `print is for printing anything back to the screen.
For many years people have printed back to the screen.`,
	Args: cobra.MinimumNArgs(0),
	Run:  testLogic,
}

func testLogic(cmd *cobra.Command, args []string) {
	// TODO: Check pprof is available on path
	const shell = "/bin/bash"
	benchmarks, _ := GetBenchmarks(".")
	var err error = nil

	goPath, err := exec.LookPath("go")

	log.Println(fmt.Sprintf("Detected %d distinct benchmarks.", len(benchmarks)))
	for _, benchmark := range benchmarks {

		cpuProfileName := fmt.Sprintf("cpuprofile-%s.out", benchmark)
		cmdS := fmt.Sprintf("%s test -bench=%s -benchtime=%s -cpuprofile %s .", goPath, benchmark, benchtime, cpuProfileName)
		log.Println(fmt.Sprintf("Running benchmark %s with the following command: %s.", benchmark, cmdS))
		err := exec.Command(shell, "-c", cmdS).Run()
		if err != nil {
			log.Fatal(err)
		}
		granularityOptions := []string{"lines", "functions"}
		exportFromPprof(cpuProfileName, benchmark, granularityOptions)
	}
	coverprofile := "coverage.out"
	cmdS := fmt.Sprintf("%s test -cover -bench=. -benchtime=0.01s -coverprofile %s .", goPath, coverprofile)
	log.Println(fmt.Sprintf("Calculating the project benchmark coverage with the following command: %s.", cmdS))
	c := exec.Command(shell, "-c", cmdS)
	var outb, errb bytes.Buffer
	c.Stdout = &outb
	c.Stderr = &errb
	err = c.Run()
	if err != nil {
		log.Fatal(err)
	}

	cleaned := strings.TrimSpace(string(outb.String()))
	lines := strings.Split(cleaned, "\n")
	coverageLine := lines[len(lines)-2]
	coverageLineS := strings.Split(coverageLine, ":")
	coverageVs := strings.Split(strings.TrimSpace(coverageLineS[1]), " ")[0]
	coverageVs = coverageVs[0 : len(coverageVs)-1]
	fmt.Println(coverageLine, coverageVs)
	exportCoverage(coverageVs)
}

func exportCoverage(vs string) {
	postBody, err := json.Marshal(map[string]string{"coverage": vs})
	if err != nil {
		log.Fatalf("An Error Occured %v", err)
	}
	fmt.Println(string(postBody))

	responseBody := bytes.NewBuffer(postBody)
	endPoint := fmt.Sprintf("%s/v1/gh/%s/%s/branch/%s/graph", codeperfApiUrl, gitOrg, gitRepo, gitBranch)
	resp, err := http.Post(endPoint, "application/json", responseBody)
	//Handle Error
	if err != nil {
		log.Fatalf("An Error Occured %v", err)
	}
	defer resp.Body.Close()

	//Read the response body
	reply, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	if resp.StatusCode != 200 {
		log.Fatalf("An error ocurred while pushing cpu data to remote %s.\nEndpoint %s. Status code %d. Reply: %s", codeperfApiUrl, endPoint, resp.StatusCode, string(reply))
	}

}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.AddCommand(cmdPrint)
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)
	defaultGitOrg := ""
	defaultGitRepo := ""
	defaultGitCommit := ""
	defaultGitBranch := ""

	r, err := git.PlainOpen(".")
	if err != nil {
		log.Println("Unable to retrieve current repo git info. Use the --git-org, --git-repo, --git-branch and --git-hash to properly fill the git info.")
	} else {
		ref, _ := r.Head()
		if ref.Name().IsBranch() {
			defaultGitBranch = ref.Name().Short()
		}
		refHash := ref.Hash().String()
		defaultGitCommit = getShortHash(refHash, 7)
		remotes, _ := r.Remotes()
		if len(remotes) > 0 {
			remoteUsed := remotes[0].Config().URLs[0]
			log.Printf("Detected a total of %d remotes. Using the 1st remote url %s to retrieve git info", len(remotes), remoteUsed)
			defaultGitOrg, defaultGitRepo = fromRemoteToOrgRepo(remoteUsed)
			log.Printf("Detected the following git vars org=%s repo=%s hash=%s\n", defaultGitOrg, defaultGitRepo, defaultGitCommit)
		}
	}

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.codeperf.yaml)")
	rootCmd.PersistentFlags().BoolVar(&local, "local", false, "don't push the data to https://codeperf.io")
	rootCmd.PersistentFlags().StringVar(&gitOrg, "git-org", defaultGitOrg, "git org")
	rootCmd.PersistentFlags().StringVar(&gitRepo, "git-repo", defaultGitRepo, "git repo")
	rootCmd.PersistentFlags().StringVar(&gitCommit, "git-hash", defaultGitCommit, "git commit hash")
	rootCmd.PersistentFlags().StringVar(&gitBranch, "git-branch", defaultGitBranch, "git branch")
	rootCmd.PersistentFlags().StringVar(&localFilename, "local-filename", "profile.json", "Local file to export the json to. Only used when the --local flag is set")
	rootCmd.PersistentFlags().StringVar(&codeperfUrl, "codeperf-url", "https://codeperf.io", "codeperf URL")
	rootCmd.PersistentFlags().StringVar(&codeperfApiUrl, "codeperf-api-url", "https://api.codeperf.io", "codeperf API URL")
	rootCmd.PersistentFlags().StringVar(&benchTime, "benchtime", "10s", "benchmark time")
	rootCmd.PersistentFlags().StringVar(&bench, "bench", "", "Benchmark name")
	//rootCmd.MarkPersistentFlagRequired("bench")
}

// Abbreviate the long hash to a short hash (7 digits)
func getShortHash(hash string, ndigits int) (short string) {
	if len(hash) < ndigits {
		short = hash
	} else {
		short = hash[:ndigits]
	}
	return
}

func fromRemoteToOrgRepo(remoteUsed string) (defaultGitOrg string, defaultGitRepo string) {
	toOrg := remoteUsed[:strings.LastIndex(remoteUsed, "/")]
	defaultGitOrg = toOrg[strings.LastIndexAny(toOrg, "/:")+1:]
	repoStartPos := strings.LastIndex(remoteUsed, "/") + 1
	repoEndPos := strings.LastIndex(remoteUsed[repoStartPos:], ".")
	if repoEndPos < 0 {
		defaultGitRepo = remoteUsed[repoStartPos:]
	} else {
		defaultGitRepo = remoteUsed[repoStartPos : repoStartPos+repoEndPos]
	}
	return
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".codeperf" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".codeperf")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
