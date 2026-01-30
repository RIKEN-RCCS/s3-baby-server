// repeating-client.go

// This is mostly taken from the AWS-SDK-GO-V2 S3 examples.
//
//  - https://github.com/awsdocs/aws-doc-sdk-examples/tree/main/gov2/s3
//  - https://docs.aws.amazon.com/code-library/latest/ug/go_2_s3_code_examples.html

// MEMO: Waiter types: BucketExistsWaiter, ObjectExistsWaiter.

// Assumption: It assumes AWS-S3 routines handle context cancellation,
// and thus a wait-group finishes.

package main

import (
	//"bytes"
	"crypto/tls"
	"context"
	"errors"
	"fmt"
	"net/http"
	//"io"
	"log"
	//"os"
	"sync"
	"time"

	//awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/aws"
	//"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	//"github.com/aws/smithy-go"
)

//var aws_s3_region = "us-east-1"
var aws_s3_region = ""
var operation_timeout = 10 * time.Second

func test_with_many_buckets(n int) error {
	// It assumes the default configuration "~/.aws/config" contains
	// definitions at least: endpoint_url, aws_access_key_id, and
	// aws_secret_access_key.

	var timeout = time.Duration(60000 * time.Millisecond)
	var xport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	var c = &http.Client{
		Transport: xport,
		Timeout:   timeout,
	}

	var cfg, err1 = config.LoadDefaultConfig(context.TODO(),
		config.WithSharedConfigProfile("default"),
		config.WithDefaultRegion("us-east-1"),
		/*
		config.WithHTTPClient(awshttp.NewBuildableClient()
			.WithTransportOptions(func(xport *http.Transport) {
				xport.MaxIdleConns = 60
				xport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			}))
		*/
		config.WithHTTPClient(c))
	if err1 != nil {
		log.Fatal("config.LoadDefaultConfig failed; error=%v", err1)
	}
	//log.Printf("AWS-S3 config=%#v\n", cfg)
	//log.Printf("- AWS-S3 Credentials=%#v\n", cfg.Credentials)
	var credentials, err3 = cfg.Credentials.Retrieve(context.TODO())
	if err3 != nil {
		log.Fatal("cfg.Credentials.Retrieve failed; error=%v", err3)
	}
	log.Printf("- AWS-S3 BaseEndpoint=%#v\n", *cfg.BaseEndpoint)
	log.Printf("- AWS-S3 Region=%#v\n", cfg.Region)
	log.Printf("- AWS-S3 AccessKeyID=%#v\n", credentials.AccessKeyID)

	var client = s3.NewFromConfig(cfg)
	//log.Printf("AWS-S3 client=%#v\n", client)

	var err2 = create_many_buckets(client, "bkt", 10)
	if err2 != nil {
		log.Fatal("create_many_buckets failed; error=%v", err2)
	}
	return nil
}

func create_many_buckets(client *s3.Client, prefix string, n int) error {
	if !(n <= 2000) {
		log.Fatal("Should be n <= 2000.")
	}

	var ctx1 = context.TODO()
	var timeout = 10 * time.Minute
	var ctx2, cancel = context.WithTimeout(ctx1, timeout)
	defer cancel()

	var wg sync.WaitGroup
	for i := range n {
		var name = prefix + fmt.Sprintf("%04d", i)
		wg.Go(func() {
			var err3 = create_bucket(ctx2, client, name)
			if err3 != nil {
				cancel()
			}
		})
	}
	wg.Wait()
	return nil
}

func create_bucket(ctx context.Context, client *s3.Client, name string) error {
	//var ctx1 = context.TODO()
	var ctx2, cancel = context.WithTimeout(ctx, operation_timeout)
	defer cancel()

	var wg sync.WaitGroup

	wg.Go(func() {
		var err2 = s3.NewBucketExistsWaiter(client).Wait(
			ctx2, &s3.HeadBucketInput{Bucket: aws.String(name)},
			operation_timeout)
		if err2 != nil {
			log.Printf("Waiter on bucket=%s failed; error=%v\n", name, err2)
			return
		}
		return
	})

	var _, err1 = client.CreateBucket(ctx2, &s3.CreateBucketInput{
		Bucket: aws.String(name),
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			//LocationConstraint: types.BucketLocationConstraint(aws_s3_region),
		},
	})
	if err1 != nil {
		var owned *types.BucketAlreadyOwnedByYou
		var exists *types.BucketAlreadyExists
		if errors.As(err1, &owned) {
			log.Printf("Bucket %s already owned.\n", name)
		} else if errors.As(err1, &exists) {
			log.Printf("Bucket %s already exists.\n", name)
		} else {
			log.Printf("Create on bucket=%s failed; error=%v\n", name, err1)
		}
		return err1
	}

	wg.Wait()

	return nil
}
