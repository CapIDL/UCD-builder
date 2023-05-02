#!/bin/bash

version=${1}
echo "Version is '${version}'"

if [ -n "${version}" ]; then
  if [[ ! ${version} =~ [0-9]+\.[0-9]+\.[0-9]+ ]] ; then
    echo "Bad unicode version number"
    exit -1
  fi

  eversion=$(echo ${version} | sed 's/\.[0-9][0-9]*$//' )
  echo "eversion is '${eversion}'"

  if [ -d "${version}" ];then
    echo "Data for version ${version} already fetched"
    exit -1
  fi

  ucdpath=zipped/${version}/
  ucdxmlpath=${version}/ucdxml/
  emojipath=emoji/${eversion}/
else
  version=latest
  eversion=latest
  ucdpath=zipped/${version}/
  ucdxmlpath="UCD/${version}/ucdxml/"
  emojipath="emoji/${eversion}/"
fi

# For now, suppress fetch of ucdxml and emojipath - I want to clean up the
# script to make these optional, and I'm not using them yet in any case.
ucdxmlpath=""
emojipath=""


wget -r -c -nH -np --reject 'index.html*,Read*' --cut-dirs=3 -P${version}/ucd https://www.unicode.org/Public/${ucdpath}
if [ -n "${ucdxmlpath}" ]; then
  wget -r -c -nH -np --reject 'index.html*' --cut-dirs=3 -P${version}/ucdxml https://www.unicode.org/Public/${ucdxmlpath}
fi
if [ -n "${emojipath}" ]; then
  wget -r -c -nH -np --reject 'index.html*' --cut-dirs=3 -Pemoji-${eversion} https://www.unicode.org/Public/${emojipath}
fi

#find . -name 'index.html*' -exec rm {} \;
find . -name 'robots.txt' -exec rm {} \;

# For reasons I do not understand, fetching the latest ucdxml places the result
# in a subdirectory, while all of the others behave. Fix it manually:
if [ -d ${version}/ucdxml/ucdxml ]; then
  mv ${version}/ucdxml/ucdxml/* ${version}/ucdxml/
  rmdir ${version}/ucdxml/ucdxml
fi

(cd ${version}/ucd
  unzip UCD.zip
  rm UCD.zip

  if [ -r Unihan.zip ]; then
    unzip -d unihan Unihan.zip
    rm Unihan.zip
  fi
)

if [ -n "${ucdxmlpath}" ]; then
  (cd ${version}/ucdxml
    for f in *.zip
    do
      unzip $f
      rm $f
    done
  )
fi