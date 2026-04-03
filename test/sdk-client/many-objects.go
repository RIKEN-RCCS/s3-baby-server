// Test by making many objects.

package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	//"github.com/aws/aws-sdk-go-v2/service/s3/types"
	//smithy "github.com/aws/smithy-go"
)

// DATASET is a bin of randomly sized random data.
var dataset [][]byte

func test_with_many_objects(cfg *aws.Config, client *s3.Client, n int) error {
	log.Printf("Testing upload/download objects n=%d\n", n)
	var bucket = "mybucket1"

	var xclient = transfermanager.New(client, func(o *transfermanager.Options) {
		//o.PartSizeBytes = part_size
	})

	var ctx1 = context.TODO()

	var err1 = control_server("stat", cfg)
	if err1 != nil {
		log.Fatalf("control_server(stat) failed; error=%v", err1)
	}

	var err2 = op_create_bucket(ctx1, client, bucket)
	if err2 != nil {
		//log.Fatalf("create-bucket failed; error=%v", err2)
	}

	var count = 20
	var size int64 = (20 * 1024 * 1024)
	var err11 = prepare_dataset(count, size)
	if err11 != nil {
		log.Fatalf("prepare_dataset failed; count=%d size=%d error=%v",
			count, size, err11)
	}

	var err3 = create_many_objects(xclient, client, bucket, "data", n)
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

// PREPARE_DATASET fills DATASET with n randomly sized data, in range
// of bytes from 100 to ub+100.
func prepare_dataset(n int, ub int64) error {
	dataset = make([][]byte, n)
	for i := range n {
		var s = rand.Int63n(ub) + 100
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

func create_many_objects(xclient *transfermanager.Client, client *s3.Client, bucket, prefix string, n int) error {
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
			var err3 = op_upload_object(ctx2, xclient, client, bucket, object, data1)
			if err3 != nil {
				cancel()
				log.Fatalf("op_upload_object() failed; error=%v", err3)
				return
			}
			var data2, err4 = op_download_object(ctx2, xclient, client, bucket, object)
			if err4 != nil {
				cancel()
				log.Fatalf("op_download_object() failed; error=%v", err4)
				return
			}
			if !bytes.Equal(data1, data2) {
				cancel()
				log.Fatalf("Unequal copy content; object=%s size1=%d size2=%d",
					object, len(data1), len(data2))
				return
			}

			// Keep an object in the bucket for a while.

			time.Sleep(10 * time.Second)

			var err5 = op_delete_one_object(ctx2, client, bucket, object)
			if err5 != nil {
				cancel()
				slog.Error("op_delete_one_object() failed",
					"object", object, "error", err5)
				return
			}
		})
	}
	wg1.Wait()

	return nil
}
