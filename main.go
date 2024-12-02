package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
)

type option struct {
	autosScalingGroupName string
	minHealthyPercentage  int
	verbose               bool
}

func main() {
	var opt option
	flag.StringVar(&opt.autosScalingGroupName, "auto-scaling-group-name", "", "Auto Scaling Group Name")
	flag.IntVar(&opt.minHealthyPercentage, "min-healthy-percentage", 100, "Minimum Healthy Percentage")
	flag.BoolVar(&opt.verbose, "verbose", false, "Verbose")
	flag.Parse()

	if opt.autosScalingGroupName == "" {
		fmt.Println("Auto Scaling Group Name is required")
		os.Exit(1)
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Printf("error loading config: %v\n", err)
		os.Exit(1)
	}

	client := autoscaling.NewFromConfig(cfg)

	minHealthyPercentage := int32(opt.minHealthyPercentage)
	startRefreshOut, err := client.StartInstanceRefresh(ctx, &autoscaling.StartInstanceRefreshInput{
		AutoScalingGroupName: &opt.autosScalingGroupName,
		Preferences: &types.RefreshPreferences{
			MinHealthyPercentage: &minHealthyPercentage,
		},
	})
	if err != nil {
		fmt.Printf("error starting instance refresh: %v\n", err)
		os.Exit(1)
	}

	for {
		rOut, _err := client.DescribeInstanceRefreshes(ctx, &autoscaling.DescribeInstanceRefreshesInput{
			AutoScalingGroupName: &opt.autosScalingGroupName,
			InstanceRefreshIds:   []string{*startRefreshOut.InstanceRefreshId},
		})
		if _err != nil {
			fmt.Printf("error describing instance refresh: %v\n", _err)
			os.Exit(1)
		}

		if len(rOut.InstanceRefreshes) == 0 {
			fmt.Println("instance refresh not found")
			os.Exit(1)
		}

		status := rOut.InstanceRefreshes[0].Status
		if status == types.InstanceRefreshStatusSuccessful {
			break
		} else if status == types.InstanceRefreshStatusFailed {
			fmt.Println("instance refresh failed")
			os.Exit(1)
		} else if status == types.InstanceRefreshStatusCancelled {
			fmt.Println("instance refresh cancelled")
			os.Exit(1)
		}

		time.Sleep(5 * time.Second)
		if opt.verbose {
			fmt.Printf("instance refresh status: %s\n", status)
		}
	}

	o, err := json.Marshal(startRefreshOut)
	if err != nil {
		fmt.Printf("error marshalling output: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(o))
}
