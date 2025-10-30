# makefile

all::
	@echo "Usage: make download-srfi-180"

# Download "srfi-180" (json) which is not avaialble in Guile-3.0.10
# yet.  We use the latest one commit="725372e" on 2023-11-19.
# commit="725372ebe5dff447188656754cf84c7ba5155cc7".  See
# https://github.com/scheme-requests-for-implementation/srfi-180

download-srfi-180::
	wget https://raw.githubusercontent.com/scheme-requests-for-implementation/srfi-180/725372ebe5dff447188656754cf84c7ba5155cc7/srfi/180/body.scm -O srfi-180-body.scm
