#!/bin/bash

DST_DIR=./export
INDEX_FILE=${DST_DIR}/index.html
ABOUT_DIR=${DST_DIR}/about
ABOUT_FILE=${ABOUT_DIR}/index.html
POST_DIR=${DST_DIR}/post
STATIC_FILE_DIR=../themes/simple
IMAGE_DIR=/home/smart/downloads/blog/source/image
HOST="http://localhost:80"

function init(){
    mkdir -p ${ABOUT_DIR}
    mkdir -p ${POST_DIR}
}

function download_post(){
    http ${HOST} > ${INDEX_FILE}
    cat ${INDEX_FILE} | grep "tag=\"export\"" | grep -oP "/post/.*?\"" | sed 's/"//g' | while read line
    do
        #not dir
        path=${DST_DIR}$line
        echo "now save "$path
        mkdir -p $path
        http ${HOST}$line > $path/index.html </dev/null

        cat $path/index.html | grep "tag=\"export\"" | grep -oP "/post/.*/.*?\"" | sed 's/"//g' | while read subline
        do
            path=${DST_DIR}$subline
            echo "now save "$path
            mkdir -p $path
            http ${HOST}$subline > $path/index.html </dev/null
        done
    done
}

function download_about(){
    http ${HOST}/about > ${ABOUT_FILE}
}

function copy_static_path(){
    cp -r ${STATIC_FILE_DIR}/css ${DST_DIR}
    cp -r ${STATIC_FILE_DIR}/js ${DST_DIR}
    cp -r ${IMAGE_DIR} ${DST_DIR}
}

init
download_post
download_about
copy_static_path
