# test/minima

It requires guile-3.0.9 or as it uses "spawn" to execute subprocesses.

## Running AWS CLI

- AWS CLI accesses "http://169.254.169.254/latest/api/token" for
  metadata (?).  To disable metadata service request, set an
  enviroment variable as:
```
export AWS_EC2_METADATA_DISABLED=true
```

## artifact-bottom.json

- Testing the "bottom" set needs to start with an empty bucket-pool.
- Bucket-pool may contain dot files (e.g., ".something").

### Note

Bucket owner should be something like
"Owner": {
    "DisplayName": "minio",
    "ID": "02d6176db174dc93cb1b899f7c6078f08654445fe8cf1b6ce98d8855f66bdbf4"
}

## Note

In AWC CLI, the "s3" command returns a non-json string, while the
"s3api" command returns json.  Note "--output json" does not work.

## json Pattern Matching

Values are one of the following data types in json, 

- string
- number
- object
- array
- boolean
- null

## TODO: CHECK ERROR CASES

- CompleteMultipartUpload operation: "EntityTooSmallError"
