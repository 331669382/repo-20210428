#coding:utf-8
#!/usr/bin/env python3

import yaml
import os
import rospy
import json
import time
from std_msgs.msg import String

qrcode = ['netcfg',132312,'F',[["wifi","wifiwifi637",False],["","",False]],[]]
cfg = {}
curPath = os.path.dirname(os.path.realpath(__file__))

def readconfig():
  yamlPath = os.path.join(curPath, "config.yaml")
  with open(yamlPath, 'r') as f:
    return yaml.load(f.read())

def subscribe():
  Last = ''
  Path = os.path.join(curPath, "LastedQRcode.txt")
  with open(Path, 'r') as f:
    Last += f.read()
  if Last!= '':
    Last = json.load(Last)
  def callback(msg):
    print('接收到/qrscan')
    try:
      jsonStr = json.load(msg.data)
    except:
      print('load JSON failed')
    else:
      # qr码的时间大于上次扫描的
      if Last=='' or Last[1] < jsonStr[1]:
        global qrcode
        qrcode = jsonStr
        with open(Path, 'w') as f:
          f.write(msg)
        configNetplan()
        GPIONotice()
      else:
        print("qr码的时间小于上次扫描的")
  rospy.init_node('netcfg')
  sub = rospy.Subscriber('qrscan', String, callback)
  rospy.spin()

def configNetplan():
  os.system('sudo chmod 777 /etc/netplan/netcfg.yaml')
  with open('/etc/netplan/netcfg.yaml', 'w') as f:
    # oldcfg = f.read()
    # netcfg = {}
    # if oldcfg != '':
    #   netcfg = yaml.load(oldcfg)
    #   netcfg['network']['wifis'] = {}
    # else:
    try:
      if qrcode[0] == 'netcfg':
        cfg1 = cfg['wifiDev'][0]
        cfg2 = cfg['wifiDev'][1]
        ssid1 = qrcode[3][0][0]
        password1 = qrcode[3][0][1]
        hidden1 = qrcode[3][0][2]
        ssid2 = qrcode[3][1][0]
        password2 = qrcode[3][1][1]
        hidden2 = qrcode[3][1][2]

        netcfg = {
          'network': {
            'version': 2,
            'renderer': "NetworkManager",
            'wifis': {
                cfg1: {
                  'dhcp4': 'yes',
                  'dhcp6': 'yes',
                  'optional': True,
                  'ipv6-privacy': False,
                  'access-points': {
                    ssid1:{}
                  }
                },
                cfg2:{
                  'dhcp4': 'yes',
                  'dhcp6': 'yes',
                  'optional': True,
                  'ipv6-privacy': False,
                  'access-points': {}
                }
            }
          }
        }

        if password1 != '':
          temp = netcfg['network']['wifis'][cfg1]['access-points'][ssid1]
          temp['password'] = password1
        if hidden1==True:
          temp['hidden'] = True


        if ssid2 != '':
          temp = netcfg['network']['wifis'][cfg2]['access-points']
          temp[ssid2] = {}
          if password2 != '':
            temp[ssid2]['password'] = password2
          if hidden2==True:
            temp['hidden'] = True
        yaml.dump(netcfg,f)
        if os.system('sudo netplan apply')!=0:
          print('网络配置失败')
        else:
          print('网络配置成功')
      except:
        print('请检查二维码')
      else:
        print('请检查二维码')

def GPIONotice():
  ##LED灯通知设置netplan成功
  print('LED')

def main():
  print('开始配置......')
  global cfg
  cfg = readconfig()
  subscribe()

if __name__ == "__main__":
  main()

