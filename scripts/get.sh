#!/bin/bash
PATH=/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin:~/bin
export PATH

#=================================================
#	System Required: Linux
#	Description: Backrest Helper
#	Author: Nebulosa Cat
#	WebSite: https://nebulosa-cat.moe
#=================================================

userHomeDir=$( getent passwd "$USER" | cut -d: -f6 )

#Coler the Bash output
Green_font_prefix="\033[32m" && Red_font_prefix="\033[31m" && Green_background_prefix="\033[42;37m" && Red_background_prefix="\033[41;37m" && Font_color_suffix="\033[0m" && Yellow_font_prefix="\033[0;33m"
Info="${Green_font_prefix}[Info]${Font_color_suffix}"
Error="${Red_font_prefix}[Error]${Font_color_suffix}"
Warring="${Yellow_font_prefix}[Warring]${Font_color_suffix}"

# Check Kernal Type
sysArch() {
    uname=$(uname -m)
    if [[ "$uname" == "x86_64" ]]; then
        arch="x86_64"
        elif [[ "$uname" == *"armv7"* ]] || [[ "$uname" == "armv6l" ]]; then
        arch="armv6"
        elif [[ "$uname" == *"armv8"* ]] || [[ "$uname" == "aarch64" ]]; then
        arch="arm64"
    else
        echo "${Error} Not Support Kernal" && exit 1
    fi
}

Install() {
    clear
    echo -e "${Info} Checking Kernal Type..."
    sysArch
    echo -e "${Info} Your Kernel Type is ${arch}."
    cd "${userHomeDir}"
    echo -e "${Info} Downloading Target Version..."
    clear
    wget https://github.com/garethgeorge/backrest/releases/latest/download/backrest_Linux_${arch}.tar.gz
    rm -r ./backrest
    mkdir ./backrest
    tar -xzvf backrest_Linux_${arch}.tar.gz -C ./backrest
    cd backrest
    clear
    echo -e "${Info} Starting Install Script..."
    ./install.sh
    echo -e "${Info} Clear Install file..."
    cd "${userHomeDir}"
    rm -r ./backrest
    rm backrest_Linux_${arch}.tar.gz
    echo -e "${Info} Install Completed!"
}

Uninstall(){
    clear
    echo -e "${Info} Checking Kernal Type..."
    sysArch
    echo -e "${Info} Your Kernel Type is ${arch}."
    cd "${userHomeDir}"
    echo -e "${Info} Downloading Target Version..."
    clear
    wget https://github.com/garethgeorge/backrest/releases/latest/download/backrest_Linux_${arch}.tar.gz
    rm -r ./backrest
    mkdir ./backrest
    tar -xzvf backrest_Linux_${arch}.tar.gz -C ./backrest
    cd backrest
    clear
    echo -e "${Info} Starting Uninstall Script..."
    ./uninstall.sh
    echo -e "${Info} Clear Install file..."
    cd "${userHomeDir}"
    rm -r ./backrest
    rm backrest_Linux_${arch}.tar.gz
    echo -e "${Info} Uninstall Completed!"
}

Start(){
    sudo systemctl start backrest
}

Stoper(){
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
 ${Red_font_prefix} Warring: This Script Only Work On Linux !${Font_color_suffix}
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
    read -e -p " Please Input [0-9]:" num
    case "$num" in
        0)
            Install
        ;;
        1)
            uninstall
        ;;
        2)
            Start
        ;;
        3)
            Stoper
        ;;
        4)
            Status
        ;;
        5)
            exit 1
        ;;
        6)
            exit 1
        ;;
        7)
            exit 1
        ;;
        8)
            exit 1
        ;;
        9)
            exit 1
        ;;
        *)
            echo "Please input correct number ${Yellow_font_prefix}[0-9]${Font_color_suffix}"
        ;;
    esac
}
Start_Menu