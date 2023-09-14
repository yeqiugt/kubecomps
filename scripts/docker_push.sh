#!/bin/bash

set -o errexit
set -o pipefail

if [ "$DEBUG" == "true" ]; then
    set -ex ;export PS4='+(${BASH_SOURCE}:${LINENO}): ${FUNCNAME[0]:+${FUNCNAME[0]}(): }'
fi

readlink_mac() {
  cd `dirname $1`
  TARGET_FILE=`basename $1`

  # Iterate down a (possible) chain of symlinks
  while [ -L "$TARGET_FILE" ]
  do
    TARGET_FILE=`readlink $TARGET_FILE`
    cd `dirname $TARGET_FILE`
    TARGET_FILE=`basename $TARGET_FILE`
  done

  # Compute the canonicalized name by finding the physical path
  # for the directory we're in and appending the target file.
  PHYS_DIR=`pwd -P`
  REAL_PATH=$PHYS_DIR/$TARGET_FILE
}

get_current_arch() {
    local current_arch
    case $(uname -m) in
        x86_64)
            current_arch=amd64
            ;;
        aarch64)
            current_arch=arm64
            ;;
    esac
    echo $current_arch
}

pushd $(cd "$(dirname "$0")"; pwd) > /dev/null
readlink_mac $(basename "$0")
cd "$(dirname "$REAL_PATH")"
CUR_DIR=$(pwd)
SRC_DIR=$(cd .. && pwd)
popd > /dev/null

DOCKER_DIR="$SRC_DIR/build/docker"

# https://docs.docker.com/develop/develop-images/build_enhancements/
export DOCKER_BUILDKIT=1

REGISTRY=${REGISTRY:-docker.io/yunion}
TAG=${TAG:-latest}
CURRENT_ARCH=$(get_current_arch)
ARCH=${ARCH:-$CURRENT_ARCH}

build_bin() {
    local BUILD_ARCH=$2
    local BUILD_CGO=$3
    case "$1" in
        *)
            docker run --rm \
                -v $SRC_DIR:/root/go/src/yunion.io/x/kubecomps \
                -v $SRC_DIR/_output/alpine-build:/root/go/src/yunion.io/x/kubecomps/_output \
                -v $SRC_DIR/_output/alpine-build/_cache:/root/.cache \
                registry.cn-beijing.aliyuncs.com/yunionio/kube-build:3.16.0-go-1.18.2-0 \
                /bin/sh -c "set -ex; cd /root/go/src/yunion.io/x/kubecomps; $BUILD_ARCH $BUILD_CGO GOOS=linux make cmd/$1; chown -R $(id -u):$(id -g) _output"
            ;;
    esac
}


build_image() {
    local tag=$1
    local file=$2
    local path=$3
    local arch
    if [[ "$tag" == *:5000/* ]]; then
        arch=$(arch)
        case $arch in
        x86_64)
            docker buildx build -t "$tag" -f "$2" "$3" --output type=docker --platform linux/amd64
            ;;
        aarch64)
            docker buildx build -t "$tag" -f "$2" "$3" --output type=docker --platform linux/arm64
            ;;
        *)
            echo wrong arch
            exit 1
            ;;
        esac
    else
        if [[ "$tag" == *"amd64" || "$ARCH" == "" || "$ARCH" == "amd64" || "$ARCH" == "x86_64" || "$ARCH" == "x86" ]]; then
            docker buildx build -t "$tag" -f "$file" "$path" --push --platform linux/amd64
        elif [[ "$tag" == *"arm64" || "$ARCH" == "arm64" ]]; then
            docker buildx build -t "$tag" -f "$file" "$path" --push --platform linux/arm64
        else
            docker buildx build -t "$tag" -f "$file" "$path" --push
        fi
        docker pull "$tag"
    fi
}

buildx_and_push() {
    local tag=$1
    local file=$2
    local path=$3
    local arch=$4
    if [[ "$DRY_RUN" == "true" ]]; then
        echo "[$(readlink -f ${BASH_SOURCE}):${LINENO} ${FUNCNAME[0]}] return for DRY_RUN"
        return
    fi
    docker buildx build -t "$tag" --platform "linux/$arch" -f "$2" "$3" --push
    docker pull --platform "linux/$arch" "$tag"
}

get_image_name() {
    local component=$1
    local arch=$2
    local is_all_arch=$3
    local img_name="$REGISTRY/$component:$TAG"
    if [[ "$is_all_arch" == "true" || "$arch" == arm64 ]]; then
        img_name="${img_name}-$arch"
    fi
    echo $img_name
}

build_process() {
    local component=$1
    local image_name=$component
    local is_all_arch=$3
    local img_name=$(get_image_name $component $arch $is_all_arch)

    build_bin $component

    build_image $img_name $DOCKER_DIR/Dockerfile.$component $SRC_DIR
}

build_process_with_buildx() {
    local component=$1
    local arch=$2
    local is_all_arch=$3
    local img_name=$(get_image_name $component $arch $is_all_arch)

    build_env="GOARCH=$arch "
    if [[ "$DRY_RUN" == "true" ]]; then
        build_bin $component $build_env
        echo "[$(readlink -f ${BASH_SOURCE}):${LINENO} ${FUNCNAME[0]}] return for DRY_RUN"
        return
    fi
    case "$component" in
        kubeserver)
            buildx_and_push $img_name $DOCKER_DIR/Dockerfile.$component-mularch $SRC_DIR $arch
            ;;
        *)
            build_bin $component $build_env
            buildx_and_push $img_name $DOCKER_DIR/Dockerfile.$component $SRC_DIR $arch
            ;;
    esac
}

general_build(){
    local component=$1
    # 如果未指定，则默认使用当前架构
    local arch=${2:-$current_arch}
    local is_all_arch=$3

    build_process_with_buildx $component $arch $is_all_arch
}

make_manifest_image() {
    local component=$1
    local img_name=$(get_image_name $component "" "false")
    if [[ "$DRY_RUN" == "true" ]]; then
        echo "[$(readlink -f ${BASH_SOURCE}):${LINENO} ${FUNCNAME[0]}] return for DRY_RUN"
        return
    fi

    docker buildx imagetools create -t $img_name \
        $img_name-amd64 \
        $img_name-arm64
    docker manifest inspect ${img_name} | grep -wq amd64
    docker manifest inspect ${img_name} | grep -wq arm64
}

ALL_COMPONENTS=$(ls cmd | grep -v '.*cli$' | xargs)

if [ "$#" -lt 1 ]; then
    echo "No component is specified~"
    echo "You can specify a component in [$ALL_COMPONENTS]"
    echo "If you want to build all components, specify the component to: all."
    exit
elif [ "$#" -eq 1 ] && [ "$1" == "all" ]; then
    echo "Build all onecloud-ee docker images"
    COMPONENTS=$ALL_COMPONENTS
else
    COMPONENTS=$@
fi

cd $SRC_DIR
mkdir -p $SRC_DIR/_output

for component in $COMPONENTS; do
    if [[ $component == *cli ]]; then
        echo "Please build image for climc"
        continue
    fi
    echo "Start to build component: $component"

    case "$ARCH" in
        all)
            for arch in "arm64" "amd64"; do
                general_build $component $arch "true"
            done
            make_manifest_image $component
            ;;
        *)
            general_build $component $ARCH "false"
            ;;
    esac
done
