#!/bin/bash

curDir=$(cd "$(dirname "$0")" && pwd)
cd "${curDir}"

echo "download hugo-theme-learn"
test -d ./themes || mkdir ./themes
git clone https://github.com/matcornic/hugo-theme-learn.git ./themes/hugo-theme-learn && rm -rf themes/hugo-theme-learn/.git

echo "copy contents to docs/.build/content"
test -d ./content || mkdir ./content
cp -r ../* ./content/
rm -rf ./content/zh

if [ "$(uname)" == "Darwin" ]; then
  find ./content -name '*.md' -exec sed -i '' 's#\.\./images#/images#g' {} \;
else
  find ./content -name '*.md' -exec sed -i 's#\.\./images#/images#g' {} \;
fi


echo "copy images"
test -d ./static || mkdir ./static
mv ./content/images ./static/
