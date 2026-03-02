# makefile

all::
	echo "Please go to the parent directory and make."

fmt::
	go fmt $(SOURCEFILES)

vet::
	go vet

test-crc::
	go test -run 'TestCRC64NVME'

test-monitor::
	go test -run 'TestMonitorExclusion'

SOURCEFILES=api-action.go auxiliary1_test.go auxiliary2_test.go
SOURCEFILES+=aws-s3-error.go aws-s3-rule.go checksum.go
SOURCEFILES+=checksum_test.go chunked-reader.go copying.go deleting.go
SOURCEFILES+=fs-operation.go hodgepodge.go listing.go metainfo.go
SOURCEFILES+=monitor.go monitor_test.go range-reader.go
SOURCEFILES+=range-reader_test.go server.go sys_bsd.go sys_unix.go
SOURCEFILES+=sys_windows.go
