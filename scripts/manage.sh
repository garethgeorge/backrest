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
greenFontPrefix="\033[32m"
redFontPrefix="\033[31m"
yellowFontPrefix="\033[0;33m"
fontColorSuffix="\033[0m"
Info="${greenFontPrefix}[Info]${fontColorSuffix}"
Error="${redFontPrefix}[Error]${fontColorSuffix}"
Warning="${yellowFontPrefix}[Warning]${fontColorSuffix}"

# Check Kernal Type
sysArch() {
    echo -e "${Info} Checking OS info..."
    uname=$(uname -m)

    if [[ "$uname" == "x86_64" ]]; 
        then arch="x86_64"
    elif [[ "$uname" == "armv7" ]] || [[ "$uname" == "armv6l" ]]; 
        then arch="armv6"
    elif [[ "$uname" == "armv8" ]] || [[ "$uname" == "aarch64" ]]; 
        then arch="arm64"
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
           Backrest Manage Helper
=============================================
 ${redFontPrefix} Warning: This Script Only Work On Linux !${fontColorSuffix}
—————————————————————————————————————————————
 ${greenFontPrefix} 1. Install / Update${fontColorSuffix} Backrest
 ${greenFontPrefix} 2.${fontColorSuffix} ${redFontPrefix}Uninstall${fontColorSuffix} Backrest
—————————————————————————————————————————————
 ${greenFontPrefix} 3.${fontColorSuffix} ${greenFontPrefix}Start${fontColorSuffix} Backrest
 ${greenFontPrefix} 4.${fontColorSuffix} ${redFontPrefix}Stop${fontColorSuffix} Backrest
—————————————————————————————————————————————
 ${greenFontPrefix} 5.${fontColorSuffix} Show Backrest Status
—————————————————————————————————————————————
 ${greenFontPrefix} 0.${fontColorSuffix} Exit Script
=============================================
"
    read -p " Please Input [0-5]:" num
    case "$num" in
        1)
            Install
        ;;
        2)
            Uninstall
        ;;
        3)
            Start
        ;;
        4)
            Stop
        ;;
        5)
            Status
        ;;
        0)
            exit 1
        ;;
        *)
            echo -e "${Error} Please enter a correct number [0-9]"
            exit 1
        ;;
    esac
}
Start_Menu