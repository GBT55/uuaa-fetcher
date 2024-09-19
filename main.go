package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/hashicorp/nomad/api"
	"os"
	"time"
)

func usage() {

	msg := fmt.Sprintf("Usage:\n  %s <job|ns>\n", os.Args[0])

	fmt.Fprintln(os.Stderr, msg)

	flag.PrintDefaults()
}

func getNomadClient() *api.Client {
	// === Declare Nomad server config ===
	config := api.DefaultConfig()

	// === Create Nomad client ===
	//pretty.Println("Creating Nomad client...")
	client, err := api.NewClient(config)
	if err != nil {
		fmt.Printf("Error trying to create Nomad client: %q", err)
	}
	dc, _ := client.Agent().Datacenter()
	fmt.Printf("Fetching Nomad DC: %q\n", dc)

	return client
}

func main() {
	//flagSet := flag.NewFlagSet("type", flag.ExitOnError)
	//var name string
	//flagSet.StringVar(&name, "name", "", "usage string")

	flag.Usage = usage

	flag.Parse()

	// user needs to provide a subcommand
	// if len(flag.Args()) < 1 {
	// 	flag.Usage()
	// 	os.Exit(1)
	// }

	//datacenters := []string{"play-es", "work-es", "work-eu", "work-mx", "work-usa", "live-es", "live-eu", "live-mx", "live-usa"}

	//for dc := range datacenters {
	//	fmt.Printf("=== DC: %v ===\n", dc)

	uuaaMapsFromJob := getUuaaFromJob()
	uuaaMapsFromNs := getUuaaFromNs()

	if len(os.Args) <= 1 {
		os.Args = append(os.Args, "")
	}

	switch os.Args[1] {
	case "job":
		fmt.Println("\nuuaa from jobs (non-empty)")
		//PrettyPrint(uuaaMapsFromJob)
		for k, v := range uuaaMapsFromJob {
			if v != "" {
				fmt.Printf("  %v: %v\n", k, v)
			}
		}

	case "ns":
		fmt.Println("\nuuaa from namespaces (non-empty)")
		//PrettyPrint(uuaaMapsFromNs)
		for k, v := range uuaaMapsFromNs {
			if v != "" {
				fmt.Printf("  %v: %v\n", k, v)
			}
		}

	default:
		fmt.Println("\nuuaa from jobs (non-empty)")
		//PrettyPrint(uuaaMapsFromJob)
		for k, v := range uuaaMapsFromJob {
			if v != "" {
				fmt.Printf("  %v: %v\n", k, v)
			}
		}

		fmt.Println("\nuuaa from namespaces (non-empty)")
		//PrettyPrint(uuaaMapsFromNs)
		for k, v := range uuaaMapsFromNs {
			if v != "" {
				fmt.Printf("  %v: %v\n", k, v)
			}
		}
	}

	//fmt.Printf("=== ===\n\n\n")
	//}
}

func getUuaaFromJob() map[string]string {
	uuaaMaps := make(map[string]string)

	nomad := getNomadClient()

	// define query options and timeout
	opts := &api.QueryOptions{
		WaitTime:  5 * time.Minute,
		Namespace: "*",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	opts.WithContext(ctx)

	nomadListFieldsOpts := api.JobListFields{
		Meta: true,
	}
	nomadJobsListOpts := &api.JobListOptions{
		Fields: &nomadListFieldsOpts,
	}
	nomadJobListQueryOpts := &api.QueryOptions{
		Namespace: "*",
	}

	jobList, _, err := nomad.Jobs().ListOptions(nomadJobsListOpts, nomadJobListQueryOpts)
	if err != nil {
		fmt.Printf("\n%v trying to retrieve job list\n", err)
	}

	for _, job := range jobList {
		ns := job.Namespace
		name := job.Name
		meta := job.Meta

		//fmt.Printf("\n---HERE---\n%v  :  %v\n", job.Namespace, job.Meta["uuaa"])

		jobAndNs := fmt.Sprintf("%s/%s", ns, name)
		uuaaMaps[jobAndNs] = meta["uuaa"]
	}

	return uuaaMaps

}

func getUuaaFromNs() map[string]string {
	uuaaMaps := make(map[string]string)

	nomad := getNomadClient()

	// define query options and timeout
	opts := &api.QueryOptions{
		WaitTime:  5 * time.Minute,
		Namespace: "*",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	opts.WithContext(ctx)

	var nsWithUuaa []string
	var nsWithoutUuaa []string
	// query namespaces list
	//nsList, meta, err := nomad.Namespaces().List(opts)
	nsList, _, err := nomad.Namespaces().List(opts)
	// TODO: instead of this we could try like 5 times over 30 seconds and then fail
	if err != nil {
		fmt.Printf("\n%v trying to retrieve namespaces list\n", err)
		return uuaaMaps
	}

	// TODO: does this even make sense?
	//if opts.WaitIndex == meta.LastIndex {
	//	// If we get the same index, we just wait again
	//	continue
	//}
	//fmt.Printf("\n%v", opts.WaitIndex)
	//fmt.Printf("\n%v", meta.LastIndex)

	for _, ns := range nsList {
		// === Define ns variables ===
		nsName := ns.Name
		nsMeta := ns.Meta
		nsUuaa := nsMeta["uuaa"]

		// === Skip these namespaces === // Not needed as we will define which namespaces to skip on the metrics.json file
		//if nsName == "admin" || nsName == "utilities" || nsName == "services" || nsName == "default" {
		//	continue
		//}

		// === Report namespaces with no uuaa ===
		// === Later we will assign the KWTC UUAA to all time series which don't have an UUAA already ===
		if nsUuaa == "" {
			//fmt.Printf("\nNamespace %q has no UUAA", nsName)
			nsWithoutUuaa = append(nsWithoutUuaa, nsName)
			continue
		}
		//-- Disable verbosity --fmt.Printf("\nNamespace %q has the following UUAA: %q", nsName, nsUuaa)
		nsWithUuaa = append(nsWithUuaa, nsName)
		uuaaMaps[nsName] = nsUuaa

		//time.Sleep(30 * time.Second)
		//fmt.Printf("\n")
	}
	//fmt.Printf("\n--- Namespaces without uuaa---\n%v\n", nsWithoutUuaa)
	//fmt.Printf("\n--Here(2)---\n%v\n", nsWithoutUuaa)
	//-- Disable verbosity --fmt.Printf("\n--- Namespaces with uuaa---\n%v\n", nsWithUuaa)
	//-- Disable verbosity --fmt.Printf("\n--- Namespaces without uuaa---\n%v\n", nsWithoutUuaa)
	return uuaaMaps
}
