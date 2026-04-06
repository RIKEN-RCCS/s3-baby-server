# Trouble Shooting

## More Logging

s3-baby-server accepts `-log trace` to dump more logging.  It is not
shown in the help message as it is usually useless.

In addition, configuration options include a few settings for
debugging.  A useful option is "dump_request_header", which dumps
request headers in "trace" logs.  s3-baby-server accepts `-conf`
conf-file.  Running `s3-baby-server dump-conf` dumps the current
configuration.  To specify options, save it to a file, modify it to
make options true, and then load it with `-conf`.
