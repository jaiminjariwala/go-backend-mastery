const $=id=>document.getElementById(id);
let events=[],cursor=-1,timer=null,dirty=true,loading=false,revision=0;
const fmt=n=>Number(n.toFixed(2)).toString();
function pause(){clearInterval(timer);timer=null;$('architecture').classList.remove('running');}
function people(){const n=Math.min(4,Math.max(0,Number($('users').value)||0));$('people').replaceChildren();for(let i=0;i<n;i++){const img=document.createElement('img');img.src='/assets/user-coral.png';img.alt='';$('people').append(img);}}
function bucket(amount){
 const cap=Number($('capacity').value)||0;$('tokens').replaceChildren();
 for(let i=0;i<Math.min(20,Math.max(0,cap));i++){const token=document.createElement('span');token.className='token';token.style.opacity=Math.min(1,Math.max(0,amount-i))*.8+.2;$('tokens').append(token);}
 $('token-count').textContent=fmt(amount)+' / '+cap+' tokens';
}
function paths(){const bounds=$('architecture').getBoundingClientRect(),b=$('bucket').getBoundingClientRect(),s=document.querySelector('.rate-server img').getBoundingClientRect(),imgs=[...$('people').querySelectorAll('img')];if(!imgs.length)return;const x=n=>(n-bounds.left)/bounds.width*1000,y=n=>(n-bounds.top)/bounds.height*600,mid=y(b.top+b.height/2);$('incoming').setAttribute('d','M'+x(Math.max(...imgs.map(i=>i.getBoundingClientRect().right))+5)+' '+mid+'H'+x(b.left));$('outgoing').setAttribute('d','M'+x(b.right)+' '+mid+'H'+x(s.left+s.width*.18));}
function render(e){
 bucket(e?e.after:Number($('capacity').value)||0);
 $('allowed').textContent=e?e.accepted:0;$('rejected').textContent=e?e.rejected:0;$('checked').textContent=e?e.request:0;
 $('bucket').classList.toggle('denied',!!e&&!e.allowed);
 $('decision').textContent=e?(e.allowed?'Request #'+e.request+' allowed · one token used':'Request #'+e.request+' rejected · 429 · bucket below one token'):'Each request needs one token.';
 if(e){paths();$('incoming').classList.add('moving');$('outgoing').classList.toggle('moving',e.accepted>0);}
 else document.querySelectorAll('.flow').forEach(p=>p.classList.remove('moving'));
 $('history-section').hidden=!e;$('history').replaceChildren();
 events.slice(0,cursor+1).reverse().forEach(e=>{const row=document.createElement('tr');for(const value of [e.time_ms+' ms','#'+e.request,'+'+fmt(e.refilled),fmt(e.before),e.allowed?'200 · allowed':'429 · rejected',fmt(e.after),e.allowed?'-':e.retry_ms+' ms']){const cell=document.createElement('td');cell.textContent=value;row.append(cell);}$('history').append(row);});
}
function reset(){revision++;pause();events=[];cursor=-1;dirty=true;$('error').textContent='';$('comment').hidden=true;people();render(null);}
async function calculate(){if(loading)return false;if(!dirty)return true;loading=true;const version=revision;$('error').textContent='';try{const response=await fetch('/api/rate-limit',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({requests:Number($('users').value),interval_ms:Number($('interval').value),capacity:Number($('capacity').value),rate:Number($('rate').value)})});const data=await response.json();if(version!==revision)return false;if(!response.ok)throw Error(data.error);events=data.events;cursor=-1;dirty=false;return true;}catch(e){if(version===revision)$('error').textContent=e.message;return false;}finally{loading=false;}}
function step(){if(cursor+1>=events.length){pause();return;}cursor++;render(events[cursor]);if(cursor===events.length-1)pause();}
$('run').addEventListener('click',async()=>{pause();if(!await calculate())return;cursor=-1;render(null);$('comment-text').textContent='The bucket starts with '+$('capacity').value+' tokens and refills at '+$('rate').value+' tokens per second. One request arrives every '+$('interval').value+' ms. Each allowed request consumes one token; an empty bucket rejects with 429, without queuing. Try increasing the spacing to see more requests pass. This is one shared simulated API limit.';$('comment').hidden=false;$('architecture').classList.add('running');timer=setInterval(step,250);});
$('step').addEventListener('click',async()=>{pause();if(await calculate())step();});$('reset').addEventListener('click',reset);$('dismiss-comment').addEventListener('click',()=>{$('comment').hidden=true;});
for(const id of ['users','interval','capacity','rate'])$(id).addEventListener('change',reset);
reset();
