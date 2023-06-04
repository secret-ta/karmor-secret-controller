#!/bin/sh

echo "The arguments passed to this script are: $@"

SECRET1_USERNAME_PATH=$(cat "$SECRET1_USERNAME_PATH")
SECRET1_PASSWORD_PATH=$(cat "$SECRET1_PASSWORD_PATH")
SECRET2_USERNAME_PATH=$(cat "$SECRET2_USERNAME_PATH")
SECRET2_PASSWORD_PATH=$(cat "$SECRET2_PASSWORD_PATH")

while true; do
  echo "secret1_username: $SECRET1_USERNAME_PATH"
  echo "secret1_password: $SECRET1_PASSWORD_PATH"
  echo "secret2_username: $SECRET2_USERNAME_PATH"
  echo "secret2_password: $SECRET2_PASSWORD_PATH"
  sleep 10
done
