// Test on body truncated case.

// (Error case) Putting but body truncated.  It checks error detection
// on the case when Content-Length and the length of the body stream
// mismatch.

package main

import (
	"bytes"
	"context"
	"errors"
	"log"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	//"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	//"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	//"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithy "github.com/aws/smithy-go"
)

func test_with_body_short(cfg *aws.Config, client *s3.Client, bucket string, n int) error {
	log.Printf("Testing going short of body n=%d\n", n)

	var err1 = control_server("stat", cfg)
	if err1 != nil {
		log.Fatalf("control_server(stat) failed; error=%v", err1)
	}

	var ctx = context.TODO()
	var object = "data1.txt"
	var truesize int64 = 5000
	var statedsize int64 = 6000
	var err2 = put_body_short(ctx, client, bucket, object,
		truesize, statedsize)
	if err2 != nil {
		slog.Error("put_body_short failed", "error", err2)
		os.Exit(2)
	}

	var err3 = control_server("stat", cfg)
	if err3 != nil {
		slog.Error("control_server(stat) failed", "error", err3)
		os.Exit(2)
	}

	return nil
}

func put_body_short(ctx context.Context, client *s3.Client, bucket, object string, truesize, statedsize int64) error {
	var data = make([]byte, truesize)
	var _, err1 = rand.Read(data)
	if err1 != nil {
		log.Fatalf("rand.Read() failed; error=%v\n", err1)
		return err1
	}

	var timeout = (1 * time.Minute)
	var f = bytes.NewReader(data)
	var _, err2 = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(object),
		ContentLength: &statedsize,
		Body:          f,
	})
	if err2 != nil {
		var err3 smithy.APIError
		if errors.As(err2, &err3) && err3.ErrorCode() == "EntityTooLarge" {
			log.Printf("Put-object too large.\n")
		}
		slog.Error("PutObject() failed",
			"object", object, "size", len(data), "error", err2)
		return err2
	}
	var err4 = s3.NewObjectExistsWaiter(client).Wait(
		ctx, &s3.HeadObjectInput{Bucket: aws.String(bucket),
			Key: aws.String(object)}, timeout)
	if err4 != nil {
		slog.Error("s3.NewObjectExistsWaiter() failed",
			"object", object, "error", err4)
		return err4
	}
	return nil
}
