package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/badgerops/terraform-provider-unifi/internal/provider"
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/badgerops/unifi",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(), opts); err != nil {
		log.Fatal(err)
	}
}
