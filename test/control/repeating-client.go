// repeating-client.go

// This is a part of the command "bbs-ctl", and runs server tests.

// The code is mostly taken from the AWS-SDK-GO-V2 S3 examples.
//
//  - https://github.com/awsdocs/aws-doc-sdk-examples/tree/main/gov2/s3
//  - https://docs.aws.amazon.com/code-library/latest/ug/go_2_s3_code_examples.html

// MEMO: Waiter types: BucketExistsWaiter, ObjectExistsWaiter.

// Assumption: It assumes AWS-S3 routines handle context cancellation,
// and thus a wait-group finishes.

// MEMO: PutObjectInput.ContentType

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
	smithy "github.com/aws/smithy-go"
)

// var aws_s3_region = "us-east-1"

func test_with_many_buckets(cfg *aws.Config, n int) error {
	log.Printf("Testing create-bucket n=%d\n", n)

	var client = s3.NewFromConfig(*cfg)
	//log.Printf("AWS-S3 client=%#v\n", client)

	var err1 = control_server("stat", cfg)
	if err1 != nil {
		log.Fatal("control_server(stat) failed; error=%v", err1)
	}

	var err2 = create_many_buckets(client, "bkt", n)
	if err2 != nil {
		log.Fatal("create_many_buckets failed; error=%v", err2)
	}

	var err3 = control_server("stat", cfg)
	if err3 != nil {
		log.Fatal("control_server(stat) failed; error=%v", err3)
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

	var wg1 sync.WaitGroup
	for i := range n {
		var name = prefix + fmt.Sprintf("%04d", i)
		wg1.Go(func() {
			var err3 = op_create_bucket(ctx2, client, name)
			if err3 != nil {
				cancel()
			}
		})
	}
	wg1.Wait()

	var buckets, err4 = op_list_buckets(ctx2, client)
	if err4 != nil {
		log.Printf("List buckets failed error=%v\n", err4)
		return err4
	}
	if len(buckets) != n {
		log.Printf("Number of buckets wrong n=%d for %d\n", len(buckets), n)
		var errx = fmt.Errorf("create many buckets failed")
		return errx
	}

	var wg2 sync.WaitGroup
	for i := range n {
		var name = prefix + fmt.Sprintf("%04d", i)
		wg2.Go(func() {
			var err3 = op_delete_bucket(ctx2, client, name)
			if err3 != nil {
				cancel()
			}
		})
	}
	wg2.Wait()

	return nil
}

func op_create_bucket(ctx context.Context, client *s3.Client, name string) error {
	//var ctx1 = context.TODO()
	var operation_timeout = 30 * time.Second

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

func op_delete_bucket(ctx context.Context, client *s3.Client, name string) error {
	var _, err1 = client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(name)})
	if err1 != nil {
		var nobucket *types.NoSuchBucket
		if errors.As(err1, &nobucket) {
			log.Printf("Bucket %s does not exist.\n", name)
		} else {
			log.Printf("DeleteBucket failed bucket=%v, error=%v\n", name, err1)
		}
		return err1
	}
	var durtion = time.Minute
	var err2 = s3.NewBucketNotExistsWaiter(client).Wait(
		ctx, &s3.HeadBucketInput{Bucket: aws.String(name)}, durtion)
	if err2 != nil {
		log.Printf("BucketNotExistsWaiter failed, bucket=%s error=%v\n", name, err2)
		return err2
	} else {
		//log.Printf("Deleted %s.\n", name)
	}
	return nil
}

func op_list_buckets(ctx context.Context, client *s3.Client) ([]types.Bucket, error) {
	var buckets []types.Bucket
	var paginator = s3.NewListBucketsPaginator(client, &s3.ListBucketsInput{})
	for paginator.HasMorePages() {
		var o, err1 = paginator.NextPage(ctx)
		if err1 != nil {
			var err2 smithy.APIError
			if errors.As(err1, &err2) && err2.ErrorCode() == "AccessDenied" {
				fmt.Printf("paginator.NextPage failed, no permission\n")
			} else {
				log.Printf("paginator.NextPage failed, error=%v\n", err1)
			}
			return nil, err1
		}
		buckets = append(buckets, o.Buckets...)
	}
	return buckets, nil
}
