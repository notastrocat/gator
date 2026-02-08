#!/bin/bash

IMAGE_NAME="gator-devenv"

# Check if the Docker image exists
if [[ "$(docker images -q $IMAGE_NAME 2> /dev/null)" == "" ]]; then
  echo
  echo -e "\e[35;1;3mImage \e[4m'$IMAGE_NAME'\e[24m is not present locally; building it now...\e[0m"
  echo
  docker build -t $IMAGE_NAME . || { echo -e "\e[31;1;3mFailed to build the Docker image.\e[0m"; exit 1; }
  echo
  echo -e "\e[32;1;3mImage \e[4m'$IMAGE_NAME'\e[24m has been built successfully.\e[0m"
  echo
  echo "starting a new container with this new image..."
  echo
  docker run --rm -it \
             --name="$IMAGE_NAME" \
             -v $(pwd):/gator \
             -w /gator \
             $IMAGE_NAME \
             /bin/bash
else
  echo -e "\e[32;1;3mImage \e[4m'$IMAGE_NAME'\e[24m is present locally.\e[0m"
  echo
fi

docker run --rm -it \
           --name="$IMAGE_NAME" \
           -v $(pwd):/gator \
           -w /gator \
           $IMAGE_NAME \
           /bin/bash