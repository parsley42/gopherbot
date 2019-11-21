#!/bin/bash -e
# mkdist.sh - create a distributable .zip file

trap_handler()
{
    ERRLINE="$1"
    ERRVAL="$2"
    echo "line ${ERRLINE} exit status: ${ERRVAL}"
    exit $ERRVAL
}
trap 'trap_handler ${LINENO} $?' ERR

usage(){
	cat <<EOF
Usage: mkdist.sh

Generate distributable .zip files for the given platform, or all platforms if
no argument given.
EOF
	exit 0
}

if [ "$1" = "-h" -o "$1" = "--help" ]
then
	usage
fi

eval `go env`
PLATFORMS=${1:-linux}
COMMIT=$(git rev-parse --short HEAD)

CONTENTS="conf/ doc/ jobs/ lib/ licenses/ plugins/ resources/ robot.skel/ scripts/ tasks/ AUTHORS.txt changelog.txt LICENSE new-robot.sh README.md"

ADIR="build-archive"
mkdir -p "$ADIR/gopherbot"
cp -a $CONTENTS "$ADIR/gopherbot"

for BUILDOS in $PLATFORMS
do
	echo "Building gopherbot for $BUILDOS"
	make clean
	OUTFILE=../gopherbot-$BUILDOS-$GOARCH.zip
	rm -f $OUTFILE
	rm -f "$ADIR/gopherbot/gopherbot"
	make
	cp -a gopherbot "$ADIR/gopherbot/gopherbot"
	cd $ADIR
	echo "Creating $OUTFILE (from $(pwd))"
	zip -r $OUTFILE gopherbot/ --exclude *.swp
	tar --exclude *.swp -czf ../gopherbot-$BUILDOS-$GOARCH.tar.gz gopherbot/
	cd -
done

rm -rf "$ADIR"
