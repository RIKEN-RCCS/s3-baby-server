// Test by making many buckets.

package main

import (
	//"bytes"
	//"crypto/tls"
	"context"
	//"errors"
	"fmt"
	//"net/http"
	//"io"
	"log"
	//"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	//"github.com/aws/aws-sdk-go-v2/service/s3/types"
	//smithy "github.com/aws/smithy-go"
)

func test_with_many_buckets(cfg *aws.Config, client *s3.Client, n int) error {
	log.Printf("Testing create-bucket n=%d\n", n)

	var err1 = control_server("stat", cfg)
	if err1 != nil {
		log.Fatalf("control_server(stat) failed; error=%v", err1)
	}

	var err2 = create_many_buckets(client, "bkt", n)
	if err2 != nil {
		log.Fatalf("create_many_buckets failed; error=%v", err2)
	}

	var err3 = control_server("stat", cfg)
	if err3 != nil {
		log.Fatalf("control_server(stat) failed; error=%v", err3)
	}

	return nil
}

func create_many_buckets(client *s3.Client, prefix string, n int) error {
	if !(n <= 2000) {
		log.Fatalf("Should be n <= 2000")
	}

	var timeout = 10 * time.Minute
	var ctx1 = context.TODO()
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
