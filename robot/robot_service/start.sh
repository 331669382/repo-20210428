#!/bin/bash
# TODO:将运行功能迁移至launch/robot_service.launch
cd ../../
source ./devel/setup.bash
catkin_make -DCATKIN_WHITELIST_PACKAGES="robot_service"
roslaunch rosbridge_server rosbridge_websocket.launch &
cd ./src/robot_service/scripts
rosrun robot_service server.py
