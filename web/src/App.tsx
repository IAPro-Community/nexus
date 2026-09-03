import React from 'react';
import { NexusWorkspaceApp } from './app/NexusWorkspaceApp';
import { NexusDemoApp } from './app/NexusDemoApp';
import type { WorkspaceSurface } from './workspace/model';

function parsePopoutSurface(): WorkspaceSurface | undefined {
  const raw=new URLSearchParams(window.location.search).get('popout');
  if(!raw)return undefined;
  try{
    const parsed=JSON.parse(raw);
    if(parsed&&typeof parsed==='object'&&parsed.id&&parsed.type)return parsed as WorkspaceSurface;
    return undefined;
  }catch{
    try{
      const decoded=JSON.parse(decodeURIComponent(raw));
      if(decoded&&typeof decoded==='object'&&decoded.id&&decoded.type)return decoded as WorkspaceSurface;
      return undefined;
    }catch{return undefined;}
  }
}
export const App:React.FC=()=> new URLSearchParams(window.location.search).get('demo') === '1' ? <NexusDemoApp/> : <NexusWorkspaceApp popoutSurface={parsePopoutSurface()}/>;
