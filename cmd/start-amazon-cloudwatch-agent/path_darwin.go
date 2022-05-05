// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

//go:build darwin
// +build darwin

package main

import (
	"log"
	"github.com/aws/amazon-cloudwatch-agent/translator/config"
	"github.com/aws/amazon-cloudwatch-agent/translator/context"
)

func setCTXOS(ctx *context.Context) {
	log.Printf("[CUSTOM] path_darwin.go setCTXOS")
	ctx.SetOs(config.OS_TYPE_DARWIN)
}
