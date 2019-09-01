
cd `dirname $0`

set -e

go build

rm -fr ../tmp
./example ./ ../tmp
rm example
