# makefile

all::
	echo "Please go to the parent diretory and make."

fmt::
	go fmt $(SOURCEFILES)

TestCRC64NVME::
	go test -run 'TestCRC64NVME'

TestMonitorExclusion::
	go test -run 'TestMonitorExclusion'

SOURCEFILES=api-action.go auxiliary1_test.go auxiliary2_test.go
SOURCEFILES+=aws-s3-aide.go aws-s3-error.go aws-s3-rule.go
SOURCEFILES+=checksum.go checksum_test.go chunked-reader.go
SOURCEFILES+=copying.go deleting.go fs-operation.go fs_linux.go
SOURCEFILES+=fs_unix.go fs_windows.go hodgepodge.go listing.go
SOURCEFILES+=metainfo.go monitor.go monitor_test.go range-reader.go
SOURCEFILES+=range-reader_test.go server.go
