// repeat-copy.go

// This is a part of the command "bbs-ctl", and runs server tests.

package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	//"github.com/aws/aws-sdk-go-v2/service/s3/types"
	//smithy "github.com/aws/smithy-go"
)

// DATASET is a bin of randomly sized random data.
var dataset [][]byte

func test_with_many_objects(cfg *aws.Config, n int) error {
	log.Printf("Testing upload/download objects n=%d\n", n)
	var bucket = "mybucket1"

	var client = s3.NewFromConfig(*cfg)
	//log.Printf("AWS-S3 client=%#v\n", client)

	var ctx1 = context.TODO()

	var err1 = control_server("stat", cfg)
	if err1 != nil {
		log.Fatalf("control_server(stat) failed; error=%v", err1)
	}

	var err2 = op_create_bucket(ctx1, client, bucket)
	if err2 != nil {
		log.Fatalf("create-bucket failed; error=%v", err2)
	}

	var count = 20
	var size int64 = (20 * 1024 * 1024)
	var err11 = prepare_dataset(count, size)
	if err11 != nil {
		log.Fatalf("prepare_dataset failed; count=%d size=%d error=%v",
			count, size, err11)
	}

	var err3 = create_many_objects(client, bucket, "data", n)
	if err3 != nil {
		log.Fatalf("create_many_objects failed; error=%v", err3)
	}

	var err4 = op_delete_bucket(ctx1, client, bucket)
	if err4 != nil {
		log.Fatalf("delete-bucket failed; error=%v", err4)
	}

	var err5 = control_server("stat", cfg)
	if err5 != nil {
		log.Fatalf("control_server(stat) failed; error=%v", err5)
	}

	return nil
}

// PREPARE_DATASET makes randomly sized data, at least 100 bytes.
func prepare_dataset(m int, size int64) error {
	dataset = make([][]byte, m)
	for i := range m {
		var s = rand.Int63n(size) + 100
		var d = make([]byte, s)
		var _, err1 = rand.Read(d)
		if err1 != nil {
			log.Fatalf("rand.Read() failed; error=%v\n", err1)
			return err1
		}
		dataset[i] = d
	}
	return nil
}

func choose_data() []byte {
	var size = len(dataset)
	var i = rand.Intn(size)
	return dataset[i]
}

func create_many_objects(client *s3.Client, bucket, prefix string, n int) error {
	if !(n <= 20000) {
		log.Fatalf("Should be n <= 20000")
	}

	var timeout = 10 * time.Minute
	var ctx1 = context.TODO()
	var ctx2, cancel = context.WithTimeout(ctx1, timeout)
	defer cancel()

	var wg1 sync.WaitGroup
	for i := range n {
		var data1 = choose_data()
		var object = prefix + fmt.Sprintf("%05d", i)
		wg1.Go(func() {
			var err3 = op_upload_object(ctx2, client, bucket, object, data1)
			if err3 != nil {
				cancel()
				log.Fatalf("op_upload_object() failed; error=%v", err3)
				return
			}
			var data2, err4 = op_download_object(ctx2, client, bucket, object)
			if err4 != nil {
				cancel()
				log.Fatalf("op_upload_object() failed; error=%v", err4)
				return
			}
			if !bytes.Equal(data1, data2) {
				cancel()
				log.Fatalf("Unequal copy content; object=%s", object)
				return
			}

			// Keep an object in the bucket for a while.

			time.Sleep(10 * time.Second)

			var err5 = op_delete_one_object(ctx2, client, bucket, object)
			if err5 != nil {
				cancel()
				log.Fatalf("op_delete_one_object() failed; object=%s error=%v",
					err5)
				return
			}
		})
	}
	wg1.Wait()

	return nil
}
