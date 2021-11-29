package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/pprof/driver"
	"github.com/google/pprof/profile"
	"github.com/spf13/cobra"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

// TextItem holds a single text report entry.
type TextItem struct {
	Symbol string `json:"symbol"`
	Flat   string `json:"flat%"`
	Cum    string `json:"cum%"`
}

// TextReport holds a list of text items from the report and a list
// of labels that describe the report.
type TextReport struct {
	Items      []TextItem `json:"data"`
	TotalRows  int        `json:"totalRows"`
	TotalPages int        `json:"totalPages"`
	Labels     []string   `json:"labels"`
}

func exportLogic() func(cmd *cobra.Command, args []string) {
	granularityOptions := []string{"lines", "functions"}
	return func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			log.Fatalf("Exactly one profile file is required")
		}
		if local {
			err, finalTree := generateFlameGraph(args[0])
			if err != nil {
				log.Fatalf("An Error Occured %v", err)
			}
			postBody, err := json.Marshal(finalTree)
			if err != nil {
				log.Fatalf("An Error Occured %v", err)
			}
			fmt.Println(string(postBody))

			for _, granularity := range granularityOptions {
				err, report := generateTextReports(granularity, args[0])
				if err == nil {
					var w io.Writer
					// open output file
					localExportLogic(w, report)
				} else {
					log.Fatal(err)
				}
			}
		} else {
			for _, granularity := range granularityOptions {
				err, report := generateTextReports(granularity, args[0])
				if err == nil {
					remoteExportLogic(report, granularity)
				} else {
					log.Fatal(err)
				}
			}
			err, finalTree := generateFlameGraph(args[0])
			if err != nil {
				log.Fatalf("An Error Occured %v", err)
			}
			remoteFlameGraphExport(finalTree)
			log.Printf("Successfully published profile data")
			log.Printf("Check it at: %s/gh/%s/%s/commit/%s/bench/%s/cpu", codeperfUrl, gitOrg, gitRepo, gitCommit, bench)
		}
	}
}

func remoteFlameGraphExport(tree treeNodeSlice) {
	postBody, err := json.Marshal(tree)
	if err != nil {
		log.Fatalf("An Error Occured %v", err)
	}
	responseBody := bytes.NewBuffer(postBody)
	endPoint := fmt.Sprintf("%s/v1/gh/%s/%s/commit/%s/bench/%s/cpu/flamegraph", codeperfApiUrl, gitOrg, gitRepo, gitCommit, bench)
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
		log.Fatalf("An error ocurred while phusing data to remote %s. Status code %d. Reply: %s", codeperfApiUrl, resp.StatusCode, string(reply))
	}

}

func remoteExportLogic(report TextReport, granularity string) {
	postBody, err := json.Marshal(report)
	if err != nil {
		log.Fatalf("An Error Occured %v", err)
	}
	responseBody := bytes.NewBuffer(postBody)
	endPoint := fmt.Sprintf("%s/v1/gh/%s/%s/commit/%s/bench/%s/cpu/%s", codeperfApiUrl, gitOrg, gitRepo, gitCommit, bench, granularity)
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
		log.Fatalf("An error ocurred while phusing data to remote %s. Status code %d. Reply: %s", codeperfApiUrl, resp.StatusCode, string(reply))
	}
}

func localExportLogic(w io.Writer, report TextReport) {
	fo, err := os.Create(localFilename)
	if err != nil {
		panic(err)
	}
	// close fo on exit and check for its returned error
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()
	// make a write buffer
	w = bufio.NewWriter(fo)
	enc := json.NewEncoder(w)
	err = enc.Encode(report)
	if err != nil {
		log.Fatalf("Unable to export the profile to local json. Error: %v", err)
	} else {
		log.Printf("Succesfully exported profile to local file %s", localFilename)
	}
}

func generateFlameGraph(input string) (err error, tree treeNodeSlice) {
	f := baseFlags()

	// Read the profile from the encoded protobuf
	outputTempFile, err := ioutil.TempFile("", "profile_output")
	if err != nil {
		log.Fatalf("cannot create tempfile: %v", err)
	}
	log.Printf("Generating temp file %s", outputTempFile.Name())
	//defer os.Remove(outputTempFile.Name())
	defer outputTempFile.Close()
	f.strings["output"] = outputTempFile.Name()
	f.bools["proto"] = true
	f.bools["text"] = false
	f.args = []string{
		input,
	}
	reader := bufio.NewReader(os.Stdin)
	options := &driver.Options{
		Flagset: f,
		UI:      &UI{r: reader},
	}

	if err = driver.PProf(options); err != nil {
		log.Fatalf("cannot read pprof profile from %s. Error: %v", input, err)
		return
	}

	file, err := os.Open(outputTempFile.Name())
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	r := bufio.NewReader(file)
	profile, err := profile.Parse(r)
	if err != nil {
		log.Fatal(err)
	}
	tree = profileToFolded(profile)
	return
}

func generateTextReports(granularity string, input string) (err error, report TextReport) {
	f := baseFlags()

	// Read the profile from the encoded protobuf
	outputTempFile, err := ioutil.TempFile("", "profile_output")
	if err != nil {
		log.Fatalf("cannot create tempfile: %v", err)
	}
	defer os.Remove(outputTempFile.Name())
	defer outputTempFile.Close()
	f.strings["output"] = outputTempFile.Name()
	f.bools["text"] = true
	f.bools[granularity] = true
	f.args = []string{
		input,
	}
	reader := bufio.NewReader(os.Stdin)
	options := &driver.Options{
		Flagset: f,
		UI:      &UI{r: reader},
	}

	if err = driver.PProf(options); err != nil {
		log.Fatalf("cannot read pprof profile from %s. Error: %v", input, err)
		return
	}

	file, err := os.Open(outputTempFile.Name())
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	r := bufio.NewReader(file)
	benchTypeStr, _, _ := r.ReadLine()
	benchDurationStr, _, _ := r.ReadLine()
	benchNodesDetails, _, _ := r.ReadLine()
	benchNodesDropDetails, _, e := r.ReadLine()
	report.Labels = append(report.Labels, string(benchTypeStr))
	report.Labels = append(report.Labels, string(benchDurationStr))
	report.Labels = append(report.Labels, string(benchNodesDetails))
	report.Labels = append(report.Labels, string(benchNodesDropDetails))
	// ignore dropped
	r.ReadLine()
	// ignore header
	r.ReadLine()
	var b []byte
	for e == nil {
		b, _, e = r.ReadLine()
		s := strings.TrimSpace(string(b))
		// ignore flat
		ns := strings.SplitN(s, " ", 2)
		if len(ns) < 2 {
			continue
		}
		s = strings.TrimSpace(ns[1])
		// read flat%
		ns = strings.SplitN(s, " ", 2)
		if len(ns) < 2 {
			continue
		}
		flatPercent := ns[0]
		s = strings.TrimSpace(ns[1])
		// ignore sum
		ns = strings.SplitN(s, " ", 2)
		if len(ns) < 2 {
			continue
		}
		s = strings.TrimSpace(ns[1])
		// ignore cum
		ns = strings.SplitN(s, " ", 2)
		if len(ns) < 2 {
			continue
		}
		s = strings.TrimSpace(ns[1])
		// read cum%
		ns = strings.SplitN(s, " ", 2)
		if len(ns) < 2 {
			continue
		}
		cumPercent := ns[0]
		symbol := strings.TrimSpace(ns[1])
		report.Items = append(report.Items, TextItem{
			Symbol: symbol,
			Flat:   flatPercent,
			Cum:    cumPercent,
		})
	}
	report.TotalPages = 1
	report.TotalRows = len(report.Items)
	return
}
