#!/bin/bash
PATH=/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin:~/bin
export PATH

#=================================================
#	Description: Backrest Helper
#	Contributed-By: https://github.com/Nebulosa-Cat
#=================================================

OS=$(uname -s)
if [[ "${OS}" != "Linux" ]]; then
    echo -e "This script is only for Linux"
    exit 1
fi

# Formatting Variables
Green_font_prefix="\033[32m"
Red_font_prefix="\033[31m"
Green_background_prefix="\033[42;37m"
Red_background_prefix="\033[41;37m"
Font_color_suffix="\033[0m"
Yellow_font_prefix="\033[0;33m"
Info="${Green_font_prefix}[Info]${Font_color_suffix}"
Error="${Red_font_prefix}[Error]${Font_color_suffix}"
Warning="${Yellow_font_prefix}[Warning]${Font_color_suffix}"

# Check Kernal Type
sysArch() {
    echo -e "${Info} Checking OS info..."
    uname=$(uname -m)

    if [[ "$uname" == "x86_64" ]]; then
        arch="x86_64"
    elif [[ "$uname" == "armv7" ]] || [[ "$uname" == "armv6l" ]]; then
        arch="armv6"
    elif [[ "$uname" == "armv8" ]] || [[ "$uname" == "aarch64" ]]; then
        arch="arm64"
    else
        echo -e "${Error} Unsupported Architecture ${arch}" && exit 1
    fi

    echo -e "${Info} You are running ${arch}."
}


Install() {
    sysArch
    tempdir=$(mktemp -d)
    cd "${tempdir}"
    echo -e  "${Info} Downloading Latest Version..."
    wget https://github.com/garethgeorge/backrest/releases/latest/download/backrest_Linux_${arch}.tar.gz
    rm -r ./backrest
    mkdir ./backrest
    tar -xzvf backrest_Linux_${arch}.tar.gz -C ./backrest
    cd backrest
    echo -e "${Info} Starting Install Script..."
    ./install.sh
    echo -e "${Info} Clearing temporary directory for install..."
    rm -rf ${tempdir}
    echo -e "${Info} Install Completed!"
}

Uninstall(){
    sysArch
    tempdir=$(mktemp -d)
    cd "${tempdir}"
    echo -e "${Info} Downloading Latest Version of uninstaller..."
    wget https://github.com/garethgeorge/backrest/releases/latest/download/backrest_Linux_${arch}.tar.gz
    rm -r ./backrest
    mkdir ./backrest
    tar -xzvf backrest_Linux_${arch}.tar.gz -C ./backrest
    cd backrest
    echo -e "${Info} Starting Uninstall Script..."
    ./uninstall.sh
    echo -e "${Info} Clear temporary directory for uninstall script..."
    rm -rf ${tempdir}
    echo -e "${Info} Uninstall Completed!"
}

Start(){
    sudo systemctl start backrest
}

Stop(){
    sudo systemctl stop backrest
}

Status(){
    sudo systemctl status backrest
}

Start_Menu(){
    clear
    echo -e "
=============================================
           Backrest Install Helper
=============================================
 ${Red_font_prefix} Warning: This Script Only Work On Linux !${Font_color_suffix}
————————————————————————————————-------------
 ${Green_font_prefix} 0.${Font_color_suffix} Install / Update Backrest
 ${Green_font_prefix} 1.${Font_color_suffix} Uninstall Backrest
—————————————————————————————————------------
 ${Green_font_prefix} 2.${Font_color_suffix} Start Backrest
 ${Green_font_prefix} 3.${Font_color_suffix} Stop Backrest
—————————————————————————————————------------
 ${Green_font_prefix} 4.${Font_color_suffix} Show Backrest Status
---------------------------------------------
 ${Green_font_prefix} 9.${Font_color_suffix} Exit Script
=============================================
"
    read -p " Please Input [0-9]:" num
    case "$num" in
        0)
            Install
        ;;
        1)
            Uninstall
        ;;
        2)
            Start
        ;;
        3)
            Stop
        ;;
        4)
            Status
        ;;
        9)
            exit 1
        ;;
        *)
            echo -e "${Error} Please enter a correct number [0-9]"
            exit 1
        ;;
    esac
}
Start_Menu