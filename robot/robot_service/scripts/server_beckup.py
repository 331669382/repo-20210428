#!/usr/bin/env python
import os,subprocess,socket
from robot_service.srv import RobotService, RobotServiceResponse
import rospy
import json,yaml
gmap=0
nav=0
tf=0
cam=0
wall=0
def unicode_convert(input):
    if isinstance(input, dict):
        return {unicode_convert(key): unicode_convert(value) for key, value in input.iteritems()}
    elif isinstance(input, list):
        return [unicode_convert(element) for element in input]
    elif isinstance(input, unicode):
        return input.encode('utf-8')
    else:
        return input
def handle_robot_service(req):
    global gmap,nav,tf,cam,wall
    # os.system('source ~/test/devel/setup.bash')
    if req.name=="startGmapping":
        if gmap!=0:
            os.killpg(os.getpgid(gmap.pid),2)
        if tf!=0:
            os.killpg(os.getpgid(tf.pid),2)
        if nav!=0:
            os.killpg(os.getpgid(nav.pid),2)
        if wall!=0:
            os.killpg(os.getpgid(wall.pid),2)
            wall=0
        os.system('rosnode kill /base_to_virtual /virtual_sensor /wall_publisher')
        os.system("rosnode kill /joint_state_publisher /ls01b_v2 /move_base /robot_state_publisher /slam_gmapping /smart_car /tf2_web_republisher /tf_odom_publish")
        os.system('rosnode kill /amcl /joint_state_publisher /ls01b_v2 /map_server /move_base /robot_state_publisher /smart_car /tf_odom_publish')
        gmap= subprocess.Popen(['roslaunch bringup gmapping_bringup.launch &'],shell=True, close_fds=True, preexec_fn = os.setsid,stdout=subprocess.PIPE)
        tf= subprocess.Popen(['rosrun tf2_web_republisher tf2_web_republisher &'],shell=True, close_fds=True, preexec_fn = os.setsid,stdout=subprocess.PIPE)
        if tf!=0 and gmap!=0:
            return RobotServiceResponse("run gmapping ok!",0,[])
        else:
            return RobotServiceResponse("run gmapping error!",-1,[])
    elif req.name=='killGmapping':
        if os.system("rosnode kill /joint_state_publisher /ls01b_v2 /move_base /robot_state_publisher /slam_gmapping /smart_car /tf2_web_republisher /tf_odom_publish")==0:
	    if gmap!=0:
	        os.killpg(os.getpgid(gmap.pid),2)
                gmap=0
	    if tf!=0:
	        os.killpg(os.getpgid(tf.pid),2)
                tf=0
	    return RobotServiceResponse("kill gmapping ok!",0,[])
        else:
	    return RobotServiceResponse("kill gmapping error!",-1,[])
    elif req.name=='saveMap':
        if os.system('cd $(rospack find bringup)/build_maps && rosrun map_server map_saver -f ./%s'%req.arg)==0:
            return RobotServiceResponse("save map ok!",0,[])
        else:
            return RobotServiceResponse("save map error!",-1,[])
    elif req.name=='startNavigation':
        if gmap!=0:
            os.killpg(os.getpgid(gmap.pid),2)
        if tf!=0:
            os.killpg(os.getpgid(tf.pid),2)
        if nav!=0:
            os.killpg(os.getpgid(nav.pid),2)
        if wall!=0:
            os.killpg(os.getpgid(wall.pid),2)
            wall=0
        os.system('rosnode kill /base_to_virtual /virtual_sensor /wall_publisher')
        os.system("rosnode kill /joint_state_publisher /ls01b_v2 /move_base /robot_state_publisher /slam_gmapping /smart_car /tf2_web_republisher /tf_odom_publish")
        os.system('rosnode kill /amcl /joint_state_publisher /ls01b_v2 /map_server /move_base /robot_state_publisher /smart_car /tf_odom_publish')
        if req.arg=='':
            return RobotServiceResponse("need arg map name!",-1,[])
        else:    
            nav= subprocess.Popen(['roslaunch bringup navigation_bringup.launch map_name:=%s'%req.arg],shell=True, close_fds=True, preexec_fn = os.setsid,stdout=subprocess.PIPE)
            tf= subprocess.Popen(['rosrun tf2_web_republisher tf2_web_republisher &'],shell=True, close_fds=True, preexec_fn = os.setsid,stdout=subprocess.PIPE)
        if nav!=0 and tf!=0:
            return RobotServiceResponse("start Navigation ok!",0,[])
        else:
            return RobotServiceResponse("start Navigation error!",-1,[])
    elif req.name=='killNavigation':
        if os.system('rosnode kill /amcl /joint_state_publisher /ls01b_v2 /map_server /move_base /robot_state_publisher /smart_car /tf_odom_publish')==0:
            if nav!=0:
                os.killpg(os.getpgid(nav.pid),2)
                nav=0
            if tf!=0:
                os.killpg(os.getpgid(tf.pid),2)
                tf=0
            return RobotServiceResponse("kill Navigation ok!",0,[])
        else:
            return RobotServiceResponse("kill Navigation error!",-1,[])
    elif req.name=='findMap':
        path=os.popen('rospack find bringup').read()[:-1]
	if req.arg=='':
	    if(os.path.isdir("%s/build_maps"%path)):
	        return RobotServiceResponse("findMap ok!",0,os.listdir("%s/build_maps"%path))
	    else:
		return RobotServiceResponse("default path didn't exit,try to set path by arg!",-1,[])
	elif os.path.isdir(req.arg):
	    return RobotServiceResponse("findMap ok!",0,os.listdir(req.arg))
	else:
	    return RobotServiceResponse("arg is not a path,try to use full path",-1,[])
    elif req.name=='startWebVideo':
        os.system('rosnode kill /web_video_server')
        if cam!=0:
            os.killpg(os.getpgid(cam.pid),2)
            cam=0
        cam= subprocess.Popen(['rosrun web_video_server web_video_server _address:=%s _port:=5000 &'%req.arg],shell=True, close_fds=True, preexec_fn = os.setsid,stdout=subprocess.PIPE)
        if cam!=0:
            return RobotServiceResponse("run WebVideo ok!",0,[])
        else:
            return RobotServiceResponse("run WebVideo error!",-1,[])
    elif req.name=='killWebVideo':
        if os.system('rosnode kill /web_video_server')==0:
            if cam!=0:
                os.killpg(os.getpgid(cam.pid),2)
                cam=0
            return RobotServiceResponse("kill WebVideo ok!",0,[])
        else:
            return RobotServiceResponse("kill WebVideo error!",-1,[])
    elif req.name=='findVirtualWall':
        print req.arg
        path=os.popen('rospack find yocs_virtual_sensor').read()[:-1]
        if req.arg=='':
            return RobotServiceResponse("please send a mapName",-1,[])
        elif os.path.isdir("%s/data"%path):
            files=os.listdir("%s/data"%path)
	    result=[]
            for f in files:
                if req.arg+"_" in f:
                    result.append(f)
            return RobotServiceResponse("findVirtualWall ok!",0,result)
        else:
            return RobotServiceResponse("default path didn't exist",-1,[])
    elif req.name=='removeVirtualWall':
        if os.system('rosnode kill /base_to_virtual /virtual_sensor /wall_publisher')==0:
            if wall!=0:
                os.killpg(os.getpgid(wall.pid),2)
                wall=0
            return RobotServiceResponse("removeVirtualWall ok!",0,[])
        else:
            return RobotServiceResponse("removeVirtualWall error!",-1,[])
    elif(req.name=='buildVirtualWall'): 
        #req.arg="{\"mapName\":\"www\",\"wallsName\":\"qqq\",\"wallArray\":[{\"length\":1,\"x\":0,\"y\":0,\"z\":0,\"w\":0}]}"
        req.arg=req.arg.replace("\\","")
        print req.arg
        try:
            js=unicode_convert(json.loads(req.arg))
        except:
            return RobotServiceResponse("json parse error!",-1,[])
        print 'js:%s'%js
        print js.get('mapName')
        print type(js.get('mapName'))
        print js.get('wallsName')
        print type(js.get('mapName'))
        print js.get('wallArray')
        print type(js.get('wallArray'))
        ylist=[]
        count=0
        wlist=[]
        if (type(js.get('mapName')) not in [str,unicode] or js.get('mapName')=='' ) or (type(js.get('wallsName'))not in [str,unicode]  or js.get('wallsName')=='' ) or (type(js.get('wallArray'))!=list or len(js.get('wallArray'))== 0 ):
            return RobotServiceResponse("arg error",-1,[])
        for i in js.get('wallArray'):
            #if 1==1:
            if type(i['x']) in [float,int] and type(i['y']) in [float,int] and type(i['z']) in [float,int] and type(i['w']) in [float,int] and type(i['length']) in [float,int]:
                y={
                    "name": 'wall%d'%count, 
                    "frame_id": 'map' ,
                    "length": i['length'], 
                    "width":  0.000001,
                    "height": 1.0,
                    "pose":
                    {
                        "position":
                        { 
                            "x": i['x'], 
                            "y": i['y'], 
                            "z": 0.0,
                        },
                        "orientation":
                        {
                            "x": 0.0,
                            "y": 0.0,
                            "z": i['z'], 
                            "w": i['w'], 
                        }
                    }
                }
                ylist.append(y)
            else:
                wlist.append(count)
            count+=1
        if len(wlist)==len(js['wallArray']):
            return RobotServiceResponse("all wallArray arg error",-1,[])
        else:
            path=os.popen('rospack find yocs_virtual_sensor').read()[:-1]+"/data/"+js['mapName']+"_"+js['wallsName']+".yaml"
            f=open(path,'w')
            yaml.dump(ylist,f)
            os.system('rosnode kill /base_to_virtual /virtual_sensor /wall_publisher')
            if wall!=0:
                os.killpg(os.getpgid(wall.pid),2)
                wall=0
            print 'roslaunch yocs_virtual_sensor standalone.launch virtual_wall:=%s &'%path
            wall= subprocess.Popen(['roslaunch yocs_virtual_sensor standalone.launch virtual_wall:=%s &'%path],shell=True, close_fds=True, preexec_fn = os.setsid,stdout=subprocess.PIPE)
            if wall!=0:
                if len(wlist)==0:
                    return RobotServiceResponse("buildVirtualWall succ",0,[])
                else:
                    return RobotServiceResponse("buildVirtualWall succ, but wallArray%s arg error"%wlist,0,[])
            else:
                return RobotServiceResponse("roslaunch virtualWall error",-1,[]) 
def robot_service_server():
    rospy.init_node('robot_service_server')  
    s = rospy.Service('robot_service', RobotService, handle_robot_service)
    print "Ready to robot_service_server."
    rospy.spin()
if __name__ == "__main__":
    robot_service_server()
