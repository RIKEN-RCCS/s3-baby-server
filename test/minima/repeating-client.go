// repeating-client.go

// This is part of the command "bbs-ctl", and processes server test
// parts.

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
	//"crypto/tls"
	"context"
	"errors"
	"fmt"
	//"net/http"
	//"io"
	"log"
	//"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// var aws_s3_region = "us-east-1"
var operation_timeout = 10 * time.Second

func test_with_many_buckets(cfg *aws.Config, n int) error {
	var client = s3.NewFromConfig(*cfg)
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
		Bucket:                    aws.String(name),
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
