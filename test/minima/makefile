# makefile

all::
	python3 cj-bottom.py
	python3 cj-server.py
	python3 cj-winscp.py
	python3 cj-argument.py

guile-json-4.5.0.tar.gz::
	wget https://download.savannah.nongnu.org/releases/guile-json/guile-json-4.5.0.tar.gz

# Download "srfi-180" (json) which is not avaialble in Guile3.0 yet.
# We use the latest one commit="725372e" on 2023-11-19.
# commit="725372ebe5dff447188656754cf84c7ba5155cc7".  See
# https://github.com/scheme-requests-for-implementation/srfi-180

srfi-180::
	wget https://raw.githubusercontent.com/scheme-requests-for-implementation/srfi-180/725372ebe5dff447188656754cf84c7ba5155cc7/srfi/180/body.scm -O srfi-180-body.scm
	wget https://raw.githubusercontent.com/scheme-requests-for-implementation/srfi-180/725372ebe5dff447188656754cf84c7ba5155cc7/srfi/srfi-180.scm -O srfi-180.scm
