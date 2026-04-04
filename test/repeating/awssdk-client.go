// AWS-SDK Client

// Note the AWS-SDK examples use "feature/s3/manager" for copying
// large objects but it is deprecated.  New one is
// "feature/s3/transfermanager".

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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	//"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithy "github.com/aws/smithy-go"
)

// var region = "us-east-1"
var part_size int64 = 10 * 1024 * 1024

func op_create_bucket(ctx context.Context, client *s3.Client, bucket string) error {
	//var ctx1 = context.TODO()
	var operation_timeout = 30 * time.Second

	var ctx2, cancel = context.WithTimeout(ctx, operation_timeout)
	defer cancel()

	var _, err3 = client.CreateBucket(ctx2, &s3.CreateBucketInput{
		Bucket:                    aws.String(bucket),
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			//LocationConstraint: types.BucketLocationConstraint(region),
		},
	})
	if err3 != nil {
		var owned *types.BucketAlreadyOwnedByYou
		var exists *types.BucketAlreadyExists
		if errors.As(err3, &owned) {
			slog.Warn("Bucket already owned", "bucket", bucket)
		} else if errors.As(err3, &exists) {
			slog.Warn("Bucket already exists", "bucket", bucket)
		} else {
			slog.Error("CreateBucket() failed", "bucket", bucket,
				"error", err3)
			log.Fatal("CreateBucket() failed")
		}
		return err3
	}

	var err2 = s3.NewBucketExistsWaiter(client).Wait(
		ctx2, &s3.HeadBucketInput{Bucket: aws.String(bucket)},
		operation_timeout)
	if err2 != nil {
		slog.Error("s3.NewBucketExistsWaiter() failed", "bucket", bucket,
			"error", err2)
		log.Fatalf("s3.NewBucketExistsWaiter() failed; bucket=%s error=%v", bucket, err2)
		return err2
	}

	return nil
}

func op_delete_bucket(ctx context.Context, client *s3.Client, bucket string) error {
	var _, err1 = client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucket)})
	if err1 != nil {
		var nobucket *types.NoSuchBucket
		if errors.As(err1, &nobucket) {
			log.Printf("Bucket %s does not exist.\n", bucket)
		}
		log.Fatalf("DeleteBucket() failed; bucket=%s, error=%v",
			bucket, err1)
		return err1
	}
	var durtion = time.Minute
	var err2 = s3.NewBucketNotExistsWaiter(client).Wait(
		ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)}, durtion)
	if err2 != nil {
		log.Fatalf("BucketNotExistsWaiter() failed; bucket=%s error=%v",
			bucket, err2)
		return err2
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
				fmt.Printf("paginator.NextPage() failed; no permission\n")
			}
			log.Fatalf("paginator.NextPage() failed; error=%v", err1)
			return nil, err1
		}
		buckets = append(buckets, o.Buckets...)
	}
	return buckets, nil
}

func op_put_object(ctx context.Context, client *s3.Client, bucket, object string, data []byte) error {
	var timeout = (1 * time.Minute)
	var f = bytes.NewReader(data)
	var _, err1 = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(object),
		Body:   f,
	})
	if err1 != nil {
		var err2 smithy.APIError
		if errors.As(err1, &err2) && err2.ErrorCode() == "EntityTooLarge" {
			log.Printf("Put-object too large.\n")
		}
		log.Fatalf("PutObject() failed; object=%s size=%d error=%v",
			object, len(data), err1)
		return err1
	}
	var err3 = s3.NewObjectExistsWaiter(client).Wait(
		ctx, &s3.HeadObjectInput{Bucket: aws.String(bucket),
			Key: aws.String(object)}, timeout)
	if err3 != nil {
		log.Fatalf("s3.NewObjectExistsWaiter() failed; object=%s error=%v",
			object, err3)
		return err3
	}
	return nil
}

func op_get_object(ctx context.Context, client *s3.Client, bucket, object string) ([]byte, error) {
	var o, err1 = client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(object),
	})
	if err1 != nil {
		var nokey *types.NoSuchKey
		if errors.As(err1, &nokey) {
			log.Printf("No such object exists: %s/%s.\n", bucket, object)
		}
		log.Fatalf("GetObject() failed; object=%s error=%v", object, err1)
		return nil, err1
	}
	var data, err2 = io.ReadAll(o.Body)
	if err2 != nil {
		log.Fatalf("io.ReadAll() failed; object=%s error=%v", object, err2)
		return nil, err2
	}
	return data, nil
}

func op_upload_object(ctx context.Context, xclient *transfermanager.Client, client *s3.Client, bucket, object string, data []byte) error {
	var timeout = (3 * time.Minute)
	var f = bytes.NewReader(data)

	var _, err1 = xclient.UploadObject(ctx, &transfermanager.UploadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(object),
		Body:   f,
	})
	if err1 != nil {
		slog.Error("uploader.Upload() failed",
			"object", object, "error", err1)
		return err1
	}

	/*
		var uploader = manager.NewUploader(client, func(u *manager.Uploader) {
			u.PartSize = part_size
		})
		var _, err1 = uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(object),
			Body:   f,
		})
		if err1 != nil {
			var err2 smithy.APIError
			if errors.As(err1, &err2) && err2.ErrorCode() == "EntityTooLarge" {
				log.Fatalf("Uploading too large file")
			}
			log.Fatalf("uploader.Upload() failed; object=%s error=%v",
				object, err1)
			return err1
		}
	*/

	var err3 = s3.NewObjectExistsWaiter(client).Wait(
		ctx, &s3.HeadObjectInput{Bucket: aws.String(bucket),
			Key: aws.String(object)}, timeout)
	if err3 != nil {
		slog.Error("s3.NewObjectExistsWaiter() failed",
			"object", object, "error", err3)
		return err3
	}
	return nil
}

func op_download_object(ctx context.Context, xclient *transfermanager.Client, client *s3.Client, bucket, object string) ([]byte, error) {
	var o, err1 = xclient.GetObject(ctx, &transfermanager.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(object),
	})
	if err1 != nil {
		slog.Error("client.GetObject() failed",
			"object", object, "error", err1)
		return nil, err1
	}
	var b = bytes.NewBuffer([]byte{})
	var n, err2 = b.ReadFrom(o.Body)
	if err2 != nil {
		slog.Error("bytes.NewBuffer.ReadFrom() failed",
			"object", object, "n", n, "error", err2)
		return nil, err2
	}

	/*
		var downloader = manager.NewDownloader(client, func(d *manager.Downloader) {
			d.PartSize = part_size
		})
		var b = manager.NewWriteAtBuffer([]byte{})
		var _, err1 = downloader.Download(ctx, b, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(object),
		})
		if err1 != nil {
			log.Fatalf("downloader.Download() failed: object=%s error=%v",
				object, err1)
			return nil, err1
		}
	*/
	return b.Bytes(), nil
}

func op_delete_one_object(ctx context.Context, client *s3.Client, bucket string, object string) error {
	var timeout = (1 * time.Minute)
	var _, err1 = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(object),
	})
	if err1 != nil {
		log.Fatalf("DeleteObject() failed; object=%s error=%v",
			object, err1)
		return err1
	}
	var err2 = s3.NewObjectNotExistsWaiter(client).Wait(
		ctx, &s3.HeadObjectInput{Bucket: aws.String(bucket),
			Key: aws.String(object)}, timeout)
	if err2 != nil {
		log.Fatalf("s3.NewObjectNotExistsWaiter() failed; error=%v",
			err2)
		return err2
	}
	return nil
}

func op_delete_objects(ctx context.Context, client *s3.Client, bucket string, objects []string) error {
	var timeout = (1 * time.Minute)
	var list []types.ObjectIdentifier
	var key string
	for _, key = range objects {
		list = append(list, types.ObjectIdentifier{Key: aws.String(key)})
	}
	var o, err1 = client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(bucket),
		Delete: &types.Delete{Objects: list, Quiet: aws.Bool(true)},
	})
	if err1 != nil || len(o.Errors) > 0 {
		log.Printf("DeleteObjects errs.\n")
		if err1 != nil {
			var nobucket *types.NoSuchBucket
			if errors.As(err1, &nobucket) {
				log.Fatalf("Bucket does not exist bucket=%s", bucket)
			}
			if len(o.Errors) > 0 {
				var err2 types.Error
				for _, err2 = range o.Errors {
					log.Printf("DeleteObjects() failed; object=%s error=%s\n",
						*err2.Key, *err2.Message)
				}
			}
			log.Fatalf("DeleteObjects() failed; error=%v", err1)
			return err1
		}
	}
	var deleted types.DeletedObject
	for _, deleted = range o.Deleted {
		var err2 = s3.NewObjectNotExistsWaiter(client).Wait(
			ctx, &s3.HeadObjectInput{Bucket: aws.String(bucket),
				Key: deleted.Key}, timeout)
		if err2 != nil {
			log.Fatalf("s3.NewObjectNotExistsWaiter() failed; error=%v",
				err2)
			return err2
		}
	}
	return nil
}
