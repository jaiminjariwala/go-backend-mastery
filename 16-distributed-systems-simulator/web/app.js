const $ = id => document.getElementById(id);
let events = [], cursor = -1, timer = null, dirty = true, loading = false;
const initial = ['A','B','C'].map(name => ({name,healthy:true,service_ms:500,assigned:0,active:0,completed:0}));
let configServers = initial.map(s => ({...s}));
let serverCount=3;
function clusterPositions(count){
  if(count===1)return [{x:0,y:0}];
  const points=[];
  // Small groups surround their shared center: three form an equilateral triangle.
  if(count<=6){
    const radius=100/(2*Math.sin(Math.PI/count));
    for(let i=0;i<count;i++){const a=-Math.PI/2+i*2*Math.PI/count;points.push({x:radius*Math.cos(a),y:radius*Math.sin(a)});}
  }else{
    points.push({x:0,y:0});
    for(let ring=1;points.length<count;ring++){
      const slots=Math.min(6*ring,count-points.length),radius=100*ring;
      for(let i=0;i<slots;i++){const a=-Math.PI/2+i*2*Math.PI/slots+(ring%2?0:Math.PI/(6*ring));points.push({x:radius*Math.cos(a),y:radius*Math.sin(a)});}
    }
  }
  const cx=(Math.min(...points.map(p=>p.x))+Math.max(...points.map(p=>p.x)))/2;
  const cy=(Math.min(...points.map(p=>p.y))+Math.max(...points.map(p=>p.y)))/2;
  return points.map(p=>({x:p.x-cx,y:p.y-cy}));
}
function syncServers(){
  const n=serverCount,service=Number($('service-time').value),slow=$('slow-server').checked;
  if(!Number.isInteger(service)||service<50||service>(slow?1250:5000)){$('error').textContent=slow?'Use 50-1250 ms with the slow-server toggle (server 1 takes 4× longer).':'Use a service time of 50-5000 ms.';return false;}
  configServers=Array.from({length:n},(_,i)=>({name:i<26?String.fromCharCode(65+i):'S'+(i+1),healthy:true,service_ms:slow&&i===0?service*4:service}));
  $('server-count').value=n;$('remove-server').disabled=n===1;$('add-server').disabled=n===30;
  buildServers();return true;
}
function pause() { clearInterval(timer); timer=null; $('architecture').classList.remove('running'); }
function people() {
  const n = Number($('users').value), count = Math.min(4, Math.max(0,Math.floor(n)));
  $('people').replaceChildren();
  for(let i=0;i<count;i++){const img=document.createElement('img');img.src='/assets/user-coral.png';img.alt='';$('people').append(img);}
}
function buildServers() {
  $('servers').replaceChildren();
  const positions=clusterPositions(configServers.length);
  const radius=Math.max(110,...positions.map(p=>Math.hypot(p.x,p.y)+55));
  $('servers').style.setProperty('--icon-size',(100/(radius*2)*100)+'%');
  configServers.forEach((s,i)=>{
    const card=document.createElement('article');card.className='server';card.id='server-'+i;
    card.innerHTML='<img src="/assets/server-box.png" alt="Server '+(i+1)+'">';
    card.style.left=(50+positions[i].x/(radius*2)*100)+'%';
    card.style.top=(50+positions[i].y/(radius*2)*100)+'%';
    card.title='Server '+(i+1)+': '+(s.healthy?'healthy':'offline')+', '+s.service_ms+' ms per request';$('servers').append(card);
  });
}
function render(event) {
  const states=event?event.states:configServers.map(s=>({...s,assigned:0,active:0,completed:0}));
  let routed=0,active=0,completed=0;
  states.forEach((s,i)=>{const card=$('server-'+i);card.classList.toggle('offline',!s.healthy);card.classList.toggle('chosen',!!event&&event.server===s.name&&event.kind==='routed');card.title='Server '+(i+1)+': '+s.assigned+' assigned, '+s.active+' active, '+s.completed+' done; '+s.service_ms+' ms';routed+=s.assigned;active+=s.active;completed+=s.completed;});
  $('routed').textContent=routed;$('active').textContent=active;$('completed').textContent=completed;$('rejected').textContent=event?event.rejected:0;
  if(!event)document.querySelectorAll('.flow').forEach(path=>path.classList.remove('moving'));
  if(event && event.kind==='routed'){
    const bounds=$('architecture').getBoundingClientRect();
    const balancer=document.querySelector('.balancer img').getBoundingClientRect();
    const userRight=Math.max(...Array.from($('people').querySelectorAll('img'),img=>img.getBoundingClientRect().right));
    const px=value=>(value-bounds.left)/bounds.width*1000,py=value=>(value-bounds.top)/bounds.height*600;
    // The generated PNG has transparent margins; anchor to the visible device.
    const visibleLeft=balancer.left+balancer.width*.13,visibleRight=balancer.right-balancer.width*.13;
    const midY=py(balancer.top+balancer.height/2),startX=px(visibleRight),branchX=startX+65;
    const boxes=states.map((s,i)=>$('server-'+i).getBoundingClientRect());
    const top=py(Math.min(...boxes.map(b=>b.top+b.height/2)));
    const bottom=py(Math.max(...boxes.map(b=>b.top+b.height/2)));
    const center=(top+bottom)/2;
    const x=px(Math.min(...boxes.map(b=>b.left)));
    const ys=[Math.min(top,center-45),center,Math.max(bottom,center+45)];
    $('incoming').setAttribute('d','M'+px(userRight+5)+' '+midY+'H'+px(visibleLeft));
    $('outgoing').setAttribute('d','M'+startX+' '+midY+'H'+branchX+ys.map(y=>' M'+branchX+' '+midY+'V'+y+'H'+x).join(''));
    // Add once; event updates must not restart the continuous flow animation.
    $('incoming').classList.add('moving');$('outgoing').classList.add('moving');
  }
  $('history').replaceChildren();
  $('history-section').hidden=cursor<0;
  // Latest event first, and only events reached by playback, never future decisions.
  events.slice(0,cursor+1).reverse().slice(0,100).forEach(e=>{const row=document.createElement('tr');const server=e.states.findIndex(s=>s.name===e.server);for(const value of [e.time_ms+' ms','#'+e.request,e.kind,server<0?'-':server+1]){const cell=document.createElement('td');cell.textContent=value;row.append(cell);}$('history').append(row);});
}
function invalidate(){pause();events=[];cursor=-1;dirty=true;people();render(null);$('comment').hidden=true;}
async function calculate(){
  if(loading)return false;
  if(!dirty)return true;
  if(!syncServers())return false;
  loading=true;$('error').textContent='';
  try{const response=await fetch('/api/load-balancer',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({algorithm:$('algorithm').value,users:Number($('users').value),interval_ms:Number($('interval').value),servers:configServers})});const data=await response.json();if(!response.ok)throw new Error(data.error||'Invalid configuration');events=data.events;cursor=-1;dirty=false;return true;}catch(error){$('error').textContent=error.message;return false;}finally{loading=false;}
}
function step(){if(cursor+1>=events.length){pause();return;}cursor++;render(events[cursor]);if(cursor===events.length-1)pause();}
function showComment(){
  const users=Number($('users').value),total=configServers.length,healthy=configServers.filter(s=>s.healthy).length;
  let text=users+' people will each send one request, spaced '+$('interval').value+' ms apart. The load balancer routes them across '+healthy+' available servers out of '+total+'. ';
  if(!healthy)text+='No servers are available, so these requests will be rejected. ';
  else if(users<healthy)text+='There are more available servers than requests, so some servers will receive no work. More servers are not automatically useful. ';
  else text+='A server can process several requests concurrently in this model; requests do not wait in a queue. ';
  if($('slow-server').checked)text+='Server 1 takes '+configServers[0].service_ms+' ms per request, four times the normal service time. Compare round robin with least connections to see the difference. ';
  text+='This demo does not calculate required capacity or automatically scale servers. These are simulated instances, not EC2 machines or Lambda invocations. No cloud resources are created or billed.';
  $('comment-text').textContent=text;$('comment').hidden=false;
}
function play(){pause();showComment();$('architecture').classList.add('running');timer=setInterval(step,120);}
$('dismiss-comment').addEventListener('click',()=>{$('comment').hidden=true;});
$('run').addEventListener('click',async()=>{pause();if(await calculate()){cursor=-1;render(null);play();}});
$('step').addEventListener('click',async()=>{pause();if(await calculate())step();});
$('reset').addEventListener('click',invalidate);
for(const id of ['users','interval','algorithm'])$(id).addEventListener('change',()=>{invalidate();const least=$('algorithm').value==='least_connections';$('rule').textContent=least?'Choose the fewest active connections.':'Take turns across healthy servers.';$('explanation').textContent=least?'A slower server keeps requests active longer, so least connections often sends fewer new arrivals to it. This does not measure CPU usage or predict future work.':'Round robin distributes arrivals, not work duration. A slow server can accumulate more active requests even with similar assigned counts.';});
for(const id of ['slow-server','service-time'])$(id).addEventListener('change',()=>{invalidate();if(syncServers()){ $('error').textContent='';render(null);}});
for(const [id,delta] of [['remove-server',-1],['add-server',1]])$(id).addEventListener('click',()=>{serverCount=Math.max(1,Math.min(30,serverCount+delta));invalidate();if(syncServers()){ $('error').textContent='';render(null);}});
buildServers();people();render(null);pause();
