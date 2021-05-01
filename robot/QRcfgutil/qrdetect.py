#coding:utf-8
#!/usr/bin/env python3

import zbar
from PIL import Image as pilImage
from PIL import ImageEnhance
import rospy
from sensor_msgs.msg import Image 
from std_msgs.msg import String
import numpy as np
import time
import os

img = 0
startTime = time.time()

def publishQrscan(data):
  msg = String(data)
  pub.publish(msg)

def callback(msg):
  global img, startTime

  now = time.time()
  # 节流 时间为2秒
  if now-startTime > 2:
    print('Detecting image......')
    startTime = now
    img = np.frombuffer(msg.data, dtype=np.uint8).reshape([msg.height,msg.width,3])
    img = pilImage.fromarray(img)
    img = ImageEnhance.Contrast(img).enhance(4.0) #增加对比度
    img = img.convert('L') #灰度化
    scanner = zbar.ImageScanner()  # 获得扫描器对象
    raw = img.tobytes()  # 流转换
    image = zbar.Image(msg.width, msg.height, 'Y800', raw)
    scanner.scan(image)
    data = ''
    # 获得二维码里的数据
    for symbol in image:
        data += symbol.data
    print('Get Data is'+data)
    if data != '':
      publishQrscan(data)
    
def main():
    rospy.init_node('qrdetect')
    pub = rospy.Publisher('qrscan',String, latch = True)
    sub = rospy.Subscriber('/usb_cam/image_raw',Image,callback)
    print 'qrdetect node is ready'
    rospy.spin()

if __name__ == "__main__":
    main()






#img = ImageEnhance.Brightness(img).enhance(2.0)#增加亮度

#img = ImageEnhance.Sharpness(img).enhance(17.0)#锐利化

#img = ImageEnhance.Contrast(img).enhance(4.0)#增加对比度

#img = img.convert('L')#灰度化


