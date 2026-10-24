#!/usr/bin/env python3
"""Offline mocked checks for the human-run UMI rebind script. No cluster calls."""
import json
import os
from pathlib import Path
import subprocess
import tempfile

SCRIPT=Path(__file__).with_name('umi-stateful-rebind-4185.sh')
CASES={
 'hermes': ('pvc-5cb237dc-4a12-49ec-80c9-29477d89fa77','services','hermes-data-truenas','truenas-iscsi-retain','80Gi','hermes-umi'),
 'postgres': ('pvc-71395a3d-7d40-4303-bff3-0f3c23849fbf','hindsight','data-hindsight-postgresql-0','truenas-iscsi-retain','8Gi','hindsight-umi'),
}

MOCK='''#!/usr/bin/env python3
import os,json,sys
from pathlib import Path
args=sys.argv[1:]
if args and args[0].startswith('--kubeconfig='): args=args[1:]
if args and args[0].startswith('--request-timeout='): args=args[1:]
if args[:2] == ['-n','argocd'] and args[2:4] == ['get','application']:
 print(json.dumps({'spec':{'syncPolicy':{'automated':{'enabled':os.environ.get('AUTO')=='1'}}}}));sys.exit(0)
if args[:2] == ['-n','services'] and args[2:5] == ['get','deployment','hermes']:
 print(os.environ.get('REPLICAS','0'));sys.exit(0)
if args[:2] == ['-n','hindsight'] and args[2:4] == ['get','deployment']:
 print(os.environ.get('REPLICAS','0'));sys.exit(0)
if args[:2] == ['-n','hindsight'] and args[2:4] == ['get','statefulset']:
 print(os.environ.get('REPLICAS','0'));sys.exit(0)
if args[:3] == ['get','pods','-A']:
 print(json.dumps({'items': [] if os.environ.get('HAS_POD')!='1' else [{'metadata':{'namespace':os.environ['NAMESPACE']},'spec':{'volumes':[{'persistentVolumeClaim':{'claimName':os.environ['PVC']}}]}}]}));sys.exit(0)
if args[:2] == ['get','volumeattachments.storage.k8s.io']:
 print(json.dumps({'items': [] if os.environ.get('HAS_VA')!='1' else [{'spec':{'source':{'persistentVolumeName':os.environ['PV']}}}]}));sys.exit(0)
if args[:3] == ['get','pv','-o']:
 print(json.dumps({'items':[json.loads(Path(os.environ['PV_JSON']).read_text())]}));sys.exit(0)
if args[:2] == ['get','pv'] and '-o' in args:
 print(Path(os.environ['PV_JSON']).read_text());sys.exit(0)
if args[:2] == ['-n',os.environ['NAMESPACE']] and args[2:4] == ['get','pvc']:
 print(Path(os.environ['PVC_JSON']).read_text());sys.exit(0)
print('mock unmatched '+repr(args),file=sys.stderr);sys.exit(1)
'''

def fixture(volume, temp):
    pv,ns,pvc,sc,size,app=CASES[volume]
    iqn=f'iqn.2005-10.org.freenas.ctl:csi-{pv}-umi'
    oldpv={'apiVersion':'v1','kind':'PersistentVolume',
           'metadata':{'name':pv,'uid':'old-pv-uid','finalizers':['kubernetes.io/pv-protection'],
                       'annotations':{'pv.kubernetes.io/provisioned-by':'org.democratic-csi.truenas-iscsi'}},
           'status':{'phase':'Bound'},
           'spec':{'capacity':{'storage':size},'accessModes':['ReadWriteOnce'],
                   'persistentVolumeReclaimPolicy':'Retain','storageClassName':sc,
                   'claimRef':{'namespace':ns,'name':pvc,'uid':'old-pvc-uid'},
                   'csi':{'driver':'org.democratic-csi.truenas-iscsi','volumeHandle':pv,
                          'volumeAttributes':{'portal':'192.168.200.244:3260','iqn':iqn,'lun':'0'}}}}
    oldpvc={'apiVersion':'v1','kind':'PersistentVolumeClaim',
            'metadata':{'namespace':ns,'name':pvc,'uid':'old-pvc-uid','finalizers':['kubernetes.io/pvc-protection'],
                        'labels':{'app':'test'},'annotations':{'volume.kubernetes.io/storage-provisioner':'org.democratic-csi.truenas-iscsi'}},
            'status':{'phase':'Bound'},
            'spec':{'accessModes':['ReadWriteOnce'],'resources':{'requests':{'storage':size}},
                    'volumeMode':'Filesystem','storageClassName':sc,'volumeName':pv}}
    p=temp/'oldpv.json'; q=temp/'oldpvc.json';p.write_text(json.dumps(oldpv));q.write_text(json.dumps(oldpvc))
    config=temp/'k3s.yaml';config.write_text('fake test kubeconfig\n')
    mockdir=temp/'bin';mockdir.mkdir()
    script=mockdir/'kubectl';script.write_text(MOCK);script.chmod(0o755)
    for command in ('iscsiadm','findmnt'):
        stub=mockdir/command;stub.write_text('#!/bin/bash\nexit 21\n');stub.chmod(0o755)
    env=dict(os.environ,PV_JSON=str(p),PVC_JSON=str(q),PV=pv,PVC=pvc,NAMESPACE=ns,
             UMI_REBIND_ROOT=str(temp/'work'),UMI_REBIND_KUBECONFIG=str(config),PATH=str(mockdir)+':'+os.environ['PATH'])
    return p,q,env

def run(volume,mode,env):
    return subprocess.run(['bash',str(SCRIPT),volume,mode],env=env,text=True,capture_output=True)

import sys
selected = sys.argv[1] if len(sys.argv)>1 else None
assert selected in (None,*CASES), 'test only: hermes or postgres'
fails=0; passed=0
for volume in ([selected] if selected else CASES):
    with tempfile.TemporaryDirectory(prefix='umi4185-good-') as t:
        d=Path(t);p,q,env=fixture(volume,d)
        for mode in ('--prepare','--check'):
            r=run(volume,mode,env)
            assert r.returncode==0,(volume,mode,r.stdout,r.stderr)
        passed+=1
        print('PASS',volume,'prepare + check',flush=True)
    scenarios=[
      ('portal drift',lambda d,p,q,env: p.write_text(p.read_text().replace('192.168.200.244:3260','192.168.200.244:9999'))),
      ('Argo re-enabled',lambda d,p,q,env: env.update(AUTO='1')),
      ('VolumeAttachment exists',lambda d,p,q,env: env.update(HAS_VA='1')),
      ('forged PV candidate',lambda d,p,q,env: (d/'work'/volume/'new-pv.json').write_text((d/'work'/volume/'new-pv.json').read_text().replace('freenas.ctl:', 'freenas.ctl:wrong-'))),
    ] if volume=='hermes' else [
      ('live consumer Pod',lambda d,p,q,env: env.update(HAS_POD='1')),
      ('portal drift',lambda d,p,q,env: p.write_text(p.read_text().replace('192.168.200.244:3260','192.168.200.244:9999'))),
    ]
    for label, mutate in scenarios:
        with tempfile.TemporaryDirectory(prefix='umi4185-bad-') as t:
            d=Path(t);p,q,env=fixture(volume,d)
            r=run(volume,'--prepare',env)
            assert r.returncode==0,(volume,label,'prepare',r.stdout,r.stderr)
            mutate(d,p,q,env)
            r=run(volume,'--check',env)
            if r.returncode==0:
                fails+=1;print('FAIL non-fail-closed',volume,label,flush=True)
            else:
                passed+=1;print('PASS fail-closed',volume,label,flush=True)
assert fails==0, f'{fails} unexpected accepts'
print(f'TESTS PASS {passed}, FAIL {fails}',flush=True)
