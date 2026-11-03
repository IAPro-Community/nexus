from PIL import Image, ImageDraw, ImageFilter
from pathlib import Path
import numpy as np, cv2, hashlib, json, zipfile
ROOT=Path(__file__).resolve().parents[4]
P=ROOT/'web/public/nex/hybrid-v1'
src=Image.open(P/'nex_source_original.png').convert('RGBA')
W,H=src.size
A=np.asarray(src).copy(); base=A.copy(); masks=[]; entries=[]
modules=[('module_1',(34,639,294,920)),('module_2',(976,640,1217,927)),('module_3',(203,887,419,1097)),('module_4',(829,893,1029,1109))]
# Asymmetric crop masks intentionally mirror the real artwork, not redrawn geometry.
for name, (x0,y0,x1,y1) in modules:
    mask=Image.new('L',(W,H),0); d=ImageDraw.Draw(mask); d.ellipse((x0+1,y0+1,x1-1,y1-1),fill=255)
    m=np.asarray(mask).copy(); rgba=A.copy(); rgba[:,:,3]=(A[:,:,3].astype(np.uint16)*m//255).astype('uint8')
    # Tight crops with padding preserve glow outside spherical hardware.
    bbox=Image.fromarray(rgba,'RGBA').getbbox(); x,y,x2,y2=bbox
    crop=Image.fromarray(rgba,'RGBA').crop((x,y,x2,y2))
    crop.save(P/f'{name}.png', optimize=True)
    entries.append(dict(id=name,file=f'{name}.png',x=x,y=y,width=x2-x,height=y2-y,role='floating-module',anchor=[round((x0+x1)/2),round((y0+y1)/2)]))
    base[:,:,3]=((base[:,:,3].astype(np.uint16)*(255-m))//255).astype('uint8')
# Eye masks: preserve original visor pixels in overlay; fill under eyes with dark navy pixel texture.
# Large photorealistic highlights are outside eye masks and stay untouched.
eyes=[('eye_left',(368,412,589,584),(391,436,565,565)),('eye_right',(672,411,897,585),(697,436,873,565))]
for name,(x0,y0,x1,y1),darkrect in eyes:
    # gradient mask with 11px soft border (high fidelity at native scale)
    mfull=np.zeros((H,W),np.uint8)
    margin=12
    mfull[y0+margin:y1-margin,x0+margin:x1-margin]=255
    # Softened border with full opacity on original cyan luminous pixels.
    mfull=cv2.GaussianBlur(mfull,(0,0),5.0)
    mfull[y0+margin+6:y1-margin-6,x0+margin+6:x1-margin-6]=255
    crop_rgba=A.copy(); crop_rgba[:,:,3]=(A[:,:,3].astype(np.uint16)*mfull//255).astype('uint8')
    bbox=Image.fromarray(crop_rgba,'RGBA').getbbox(); x,y,x2,y2=bbox
    crop=Image.fromarray(crop_rgba,'RGBA').crop((x,y,x2,y2))
    crop.save(P/f'{name}.png',optimize=True)
    entries.append(dict(id=name,file=f'{name}.png',x=x,y=y,width=x2-x,height=y2-y,role='visor-expression',anchor=[(x0+x1)//2,(y0+y1)//2]))
    # Where eyes move/close, replace emitted eye light below by navy visor, without touching the metal rim.
    # Inpaint only eye core region. This remains covered at rest by the original pixel layer.
    xx,yy,xx1,yy1=darkrect
    under=base.copy(); roi=under[yy:yy1,xx:xx1,:3]; hh,ww=roi.shape[:2]
    # Smooth navy gradient sampled from dark visor, with subtle noise/scanlines.
    Y=np.arange(hh)[:,None]; X=np.arange(ww)[None,:]
    noise=((X*7+Y*11)%17)-8
    core=np.zeros_like(roi)
    core[:,:,0]=np.clip(1+Y*.025+noise*.09,0,16)
    core[:,:,1]=np.clip(6+Y*.023+noise*.16,0,25)
    core[:,:,2]=np.clip(23+Y*.075+noise*.45,10,60)
    under[yy:yy1,xx:xx1,:3]=core
    # Feather only at the core perimeter so changing eyes doesn't create a rectangular patch.
    blend=np.zeros((H,W),np.uint8)
    blend[yy+8:yy1-8,xx+8:xx1-8]=255
    blend=cv2.GaussianBlur(blend,(0,0),6)
    q=(blend.astype(np.float32)/255)[:,:,None]
    base[:,:,:3]=((1-q)*base[:,:,:3]+q*under[:,:,:3]).astype('uint8')
    # Remove upper eye-patch where opaque, while retaining original visor outside its mask.
    base[:,:,3]=(base[:,:,3].astype(np.uint16)*(255-mfull)//255).astype('uint8')
# Retouch visor underlays: feathered elliptical navy fill and restore real visor opacity.
# This prevents bright rectangular holes when the eyes blink or look away.
Y,X=np.mgrid[:H,:W]
fill=np.zeros((H,W),np.float32)
for cx,cy,rx,ry in [(479,499,160,118),(786,499,164,120)]:
    roi=np.zeros((H,W),np.float32)
    cv2.ellipse(roi,(cx,cy),(rx,ry),0,0,360,1,-1)
    fill=np.maximum(fill,cv2.GaussianBlur(roi,(0,0),20))
shade=np.zeros((H,W,3),np.float32)
edge=np.maximum(0,((X-625)/350)**2)
row=(Y-450)/210
shade[:,:,0]=np.clip(1+2*edge,0,6)
shade[:,:,1]=np.clip(5+3*edge+1.8*row,0,14)
shade[:,:,2]=np.clip(24+15*edge+2.2*row,0,60)
base[:,:,:3]=(base[:,:,:3]*(1-fill[:,:,None])+shade*fill[:,:,None]).round().clip(0,255).astype('uint8')
visibility=np.zeros((H,W),np.uint8)
for cx,cy,rx,ry in [(479,499,147,112),(786,499,149,114)]:
    cv2.ellipse(visibility,(cx,cy),(rx,ry),0,0,360,255,-1)
visibility=cv2.GaussianBlur(visibility,(0,0),12)
base[:,:,3]=np.maximum(base[:,:,3],(A[:,:,3].astype('float32')*visibility/255).astype('uint8'))

# Pixel-perfect base image as an editable raster layer.
Image.fromarray(base,'RGBA').save(P/'body_base.png',optimize=True)
# exact original is kept as fallback and canonical reference
# Preserve source bytes without any PNG re-encoding; source is already present on disk.
# Generate photorealistic fin highlight overlays without removing fin roots.
fins=[('fin_left',(228,0,433,360)),('fin_right',(830,0,1044,360))]
for name,(x0,y0,x1,y1) in fins:
    rgba=np.zeros_like(A)
    # Fin contour: upper long tapered blades, not the helmet side/ear couplings.
    mask=Image.new('L',(W,H),0);d=ImageDraw.Draw(mask)
    if name.endswith('left'):
        points=[(365,12),(407,15),(396,150),(399,270),(359,329),(296,354),(238,338),(270,269),(299,145)]
    else:
        points=[(887,12),(925,9),(966,133),(1023,279),(1038,338),(970,352),(920,317),(864,234)]
    d.polygon(points,fill=255)
    mask=mask.filter(ImageFilter.GaussianBlur(1))
    m=np.asarray(mask)
    rgba[:,:,:3]=A[:,:,:3];rgba[:,:,3]=(A[:,:,3].astype(np.uint16)*m//255).astype('uint8')
    bbox=Image.fromarray(rgba,'RGBA').getbbox(); x,y,x2,y2=bbox
    Image.fromarray(rgba,'RGBA').crop((x,y,x2,y2)).save(P/f'{name}.png', optimize=True)
    entries.append(dict(id=name,file=f'{name}.png',x=x,y=y,width=x2-x,height=y2-y,role='fin-glint',anchor=[(x0+x1)//2,y1]))
# Reconstitute from layered images to measure exact visual parity.
recomp=Image.open(P/'body_base.png').convert('RGBA')
for e in entries:
 if e['role']=='fin-glint':continue
 layer=Image.new('RGBA',(W,H));layer.alpha_composite(Image.open(P/e['file']), (e['x'],e['y']))
 recomp=Image.alpha_composite(recomp,layer)
recomp.save(P/'reconstructed_idle.png', optimize=True)
d=np.asarray(recomp).astype(np.int16)-A.astype(np.int16)
error=np.mean(np.abs(d[:,:,:3])); maxerror=np.abs(d).max()
# ORA: a real open layered master, conventional editor support (GIMP/Krita).
# Source image not replaced, visual reference remains the authoritative comparison.
import xml.etree.ElementTree as ET
root=ET.Element('image',{'w':str(W),'h':str(H),'name':'NEX hybrid layered art — fidelity review required','version':'0.0.3'})
stack=ET.SubElement(root,'stack')
ora_layers=[('Original (reference; hidden)', 'nex_source_original.png', False),('Body & visor underlayers','body_base.png',True)]
for e in entries:
 ora_layers.append((e['id'],e['file'],e['role']!='fin-glint'))
for name,file,visible in ora_layers[::-1]:
 e=next((x for x in entries if x['file']==file),None)
 ET.SubElement(stack,'layer',{'name':name,'src':'data/'+file,'opacity':'1.0','visibility':'visible' if visible else 'hidden','composite-op':'svg:src-over','x':str(e['x'] if e else 0),'y':str(e['y'] if e else 0)})
with zipfile.ZipFile(Path(__file__).parent/'nex_hybrid_master.ora','w') as z:
 z.writestr('mimetype','image/openraster',compress_type=zipfile.ZIP_STORED)
 z.writestr('stack.xml',ET.tostring(root,encoding='utf-8',xml_declaration=True))
 z.write(P/'reconstructed_idle.png','mergedimage.png')
 for _,f,_ in ora_layers:z.write(P/f,'data/'+f)
manifest={'schema_version':'1.0','status':'CANDIDATE_LAYERED_RASTER_NOT_YET_HUMAN_APPROVED','source':'nex_source_original.png','source_sha256':hashlib.sha256((P/'nex_source_original.png').read_bytes()).hexdigest(),'canvas':[W,H],'original_png_dimensions':[W,H],'base':'body_base.png','layers':entries,'source_reconstruction_mean_abs_rgb_delta':round(float(error),4),'source_reconstruction_max_channel_delta':int(maxerror),'note':'Static source file is kept unmodified. Extracted layers used for approximate motion; static parity must be visually reviewed, especially visor underlayers. No complete 360 atlas.'}
(P/'manifest.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2))
print(json.dumps({'files':[(x.name,x.stat().st_size) for x in P.iterdir() if x.is_file()],'meanDelta':error,'maxDelta':int(maxerror)},ensure_ascii=False))
