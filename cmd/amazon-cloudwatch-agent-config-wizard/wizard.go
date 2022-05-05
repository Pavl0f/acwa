// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"bufio"
	"log"
	"flag"
	"fmt"
	"os"

	"github.com/aws/amazon-cloudwatch-agent/tool/data"
	"github.com/aws/amazon-cloudwatch-agent/tool/processors"
	"github.com/aws/amazon-cloudwatch-agent/tool/processors/basicInfo"
	"github.com/aws/amazon-cloudwatch-agent/tool/processors/migration/linux"
	"github.com/aws/amazon-cloudwatch-agent/tool/processors/migration/windows"
	"github.com/aws/amazon-cloudwatch-agent/tool/processors/serialization"
	"github.com/aws/amazon-cloudwatch-agent/tool/runtime"
	"github.com/aws/amazon-cloudwatch-agent/tool/stdin"
	"github.com/aws/amazon-cloudwatch-agent/tool/testutil"
	"github.com/aws/amazon-cloudwatch-agent/tool/util"
)

type IMainProcessor interface {
	VerifyProcessor(processor interface{})
}
type MainProcessorStruct struct{}

var MainProcessorGlobal IMainProcessor = &MainProcessorStruct{}

var isNonInteractiveWindowsMigration *bool

func main() {
	log.Printf("[CUSTOM] wizard.go main")

	// Parse command line args for non-interactive Windows migration
	isNonInteractiveWindowsMigration = flag.Bool("isNonInteractiveWindowsMigration", false,
		"If true, it will use command line args to bypass the wizard. Default value is false.")

	isNonInteractiveLinuxMigration := flag.Bool("isNonInteractiveLinuxMigration", false,
		"If true, it will do the linux config migration. Default value is false.")

	useParameterStore := flag.Bool("useParameterStore", false,
		"If true, it will use the parameter store for the migrated config storage.")

	configFilePath := flag.String("configFilePath", "",
		fmt.Sprintf("The path of the old config file. Default is %s on Windows or %s on Linux", windows.DefaultFilePathWindowsConfiguration, linux.DefaultFilePathLinuxConfiguration))

	parameterStoreName := flag.String("parameterStoreName", "", "The parameter store name. Default is AmazonCloudWatch-windows")
	parameterStoreRegion := flag.String("parameterStoreRegion", "", "The parameter store region. Default is us-east-1")

	flag.Parse()

	if *isNonInteractiveWindowsMigration {
		addWindowsMigrationInputs(*configFilePath, *parameterStoreName, *parameterStoreRegion, *useParameterStore)
	} else if *isNonInteractiveLinuxMigration {
		ctx := new(runtime.Context)
		config := new(data.Config)
		ctx.HasExistingLinuxConfig = true
		ctx.ConfigFilePath = *configFilePath
		if ctx.ConfigFilePath == "" {
			ctx.ConfigFilePath = linux.DefaultFilePathLinuxConfiguration
		}
		process(ctx, config, linux.Processor, serialization.Processor)
		return
	}

	startProcessing()
}

func init() {
	log.Printf("[CUSTOM] wizard.go init")

	stdin.Scanln = func(a ...interface{}) (n int, err error) {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		if len(a) > 0 {
			*a[0].(*string) = scanner.Text()
			n = len(*a[0].(*string))
		}
		err = scanner.Err()
		return
	}
	processors.StartProcessor = basicInfo.Processor
}

func addWindowsMigrationInputs(configFilePath string, parameterStoreName string, parameterStoreRegion string, useParameterStore bool) {
	log.Printf("[CUSTOM] wizard.go addWindowsMigrationInputs")
	inputChan := testutil.SetUpTestInputStream()
	if useParameterStore {
		testutil.Type(inputChan, "2", "1", "2", "1", configFilePath, "1", parameterStoreName, parameterStoreRegion, "1")
	} else {
		testutil.Type(inputChan, "2", "1", "2", "1", configFilePath, "2")
	}
}

func process(ctx *runtime.Context, config *data.Config, processors ...processors.Processor) {
	log.Printf("[CUSTOM] wizard.go process")
	for _, processor := range processors {
		processor.Process(ctx, config)
	}
}

func startProcessing() {
	log.Printf("[CUSTOM] wizard.go startProcessing")
	ctx := new(runtime.Context)
	config := new(data.Config)

	var processor interface{}
	processor = processors.StartProcessor

	for {
		if processor == nil {
			if util.CurOS() == util.OsTypeWindows && !*isNonInteractiveWindowsMigration {
				util.EnterToExit()
			}
			fmt.Println("Program exits now.")
			break
		}
		MainProcessorGlobal.VerifyProcessor(processor) // For testing purposes
		processor.(processors.Processor).Process(ctx, config)
		processor = processor.(processors.Processor).NextProcessor(ctx, config)
	}
}

func (p *MainProcessorStruct) VerifyProcessor(processor interface{}) {
}
