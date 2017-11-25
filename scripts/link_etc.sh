#! /bin/bash

set -eu

BASE_DIR=$HOME

MYSQL_CONF_DIR=/etc/mysql
NGINX_CONF_DIR=/etc/nginx

mkdir -p $BASE_DIR/$MYSQL_CONF_DIR
mkdir -p $BASE_DIR/$NGINX_CONF_DIR

TARGETS=(
        # /etc/sysctl.conf
        # /etc/mysql/my.cnf
        # /etc/mysql/conf.d
        # /etc/mysql/mysql.conf.d
        # /etc/nginx/nginx.conf
        # /etc/nginx/conf.d
        # /etc/nginx/sites-enabled
        # /etc/nginx/sites-available
)

for target in ${TARGETS[@]}
do
  if [[ ! -e $target.backup ]]; then
    sudo cp -r $target $target.backup
    sudo mv $target $BASE_DIR/$target
    sudo ln -s $BASE_DIR/$target $target
  fi
done
