<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useId } from 'vue';
import { X, ShieldCheck, ShieldAlert, RefreshCw } from 'lucide-vue-next';
import { api } from '../api';
import type { CertificateInfo, CertificateTrust as Trust, CertificateInspection as Inspection } from '../certificates';
const props=defineProps<{
  connection:{id:string;name:string;host:string;port:number;protocol:string};
  initial?:Inspection; forConnection:boolean;
}>();
const emit=defineEmits<{close:[];ready:[check:string];changed:[]}>();
const dialog=ref<HTMLDialogElement>();
const id=useId();
const inspection=ref<Inspection|undefined>(props.initial),trust=ref<Trust|null>(props.initial?.previous||null);
const lastCertificate=ref<CertificateInfo|undefined>(props.initial?.certificate);
const busy=ref(false),error=ref(''),notice=ref(''),revokePending=ref(false);
let alive=true;
const info=computed(()=>inspection.value?.certificate||trust.value?.certificate||lastCertificate.value);
const status=computed(()=>inspection.value?.status||(trust.value?'trusted':'untrusted'));
const ready=computed(()=>inspection.value&&['valid','trusted'].includes(inspection.value.status));
const labels={valid:'证书有效',trusted:'已保存信任',untrusted:'首次确认',changed:'证书或目标已变化',invalid:'证书不可用'};
const verification:Record<string,string>={valid:'签发机构和目标名称校验通过',unknown_authority:'签发机构不在当前可信列表中',name_mismatch:'证书中的名称与连接地址不一致',expired:'证书已过期或尚未生效'};
function date(value:string){return new Date(value).toLocaleString();}
function close(){if(!busy.value)emit('close');}
async function inspect(){
  if(busy.value)return;
  busy.value=true;error.value='';notice.value='';revokePending.value=false;inspection.value=undefined;
  try {
    const result=await api<Inspection>('/connections/'+props.connection.id+'/certificate','POST',{});
    if(!alive)return;
    inspection.value=result;trust.value=result.previous;lastCertificate.value=result.certificate;
  }catch(e){if(alive)error.value=(e as Error).message;}
  finally{if(alive)busy.value=false;}
}
async function confirm(){
  if(busy.value||!inspection.value?.certificate.canTrust)return;
  busy.value=true;error.value='';
  try{
    const result=await api<{trust:Trust;check:string}>('/connections/'+props.connection.id+'/trust','POST',{check:inspection.value.check});
    if(!alive)return;
    trust.value=result.trust;inspection.value.status='trusted';inspection.value.previous=result.trust;
    emit('changed');
    if(props.forConnection)emit('ready',result.check);
    else notice.value='已保存信任，后续连接将自动核对该证书。';
  }catch(e){if(alive){error.value=(e as Error).message;inspection.value=undefined;}}
  finally{if(alive)busy.value=false;}
}
async function revoke(){
  busy.value=true;error.value='';
  try{
    await api('/connections/'+props.connection.id+'/trust','DELETE');
    if(!alive)return;
    trust.value=null;inspection.value=undefined;revokePending.value=false;
    notice.value='已撤销信任。下次连接将重新检查证书。';emit('changed');
  }catch(e){if(alive)error.value=(e as Error).message;}
  finally{if(alive)busy.value=false;}
}
onMounted(async()=>{
  await nextTick();dialog.value?.showModal();dialog.value?.querySelector<HTMLButtonElement>('[data-cancel]')?.focus();
  if(props.initial)return;
  busy.value=true;
  try{const r=await api<{trust:Trust|null}>('/connections/'+props.connection.id+'/trust');if(alive)trust.value=r.trust;}
  catch(e){if(alive)error.value=(e as Error).message;}
  finally{if(alive)busy.value=false;}
  if(alive&&props.connection.protocol==='rdp')await inspect();
});
onBeforeUnmount(()=>{alive=false;dialog.value?.close();});
</script>
<template>
  <dialog ref="dialog" class="certificate-dialog" :aria-labelledby="id+'-title'" @cancel.prevent="close" @click="e=>{if(e.target===dialog)close()}">
    <form @submit.prevent="confirm">
      <div class="dialog-heading">
        <h2 :id="id+'-title'">{{ forConnection?'确认远程电脑证书':'证书与信任' }}</h2>
        <button type="button" aria-label="关闭证书窗口" :disabled="busy" @click="close"><X :size="18"/></button>
      </div>
      <div class="dialog-body certificate-body" :aria-busy="busy">
        <div class="certificate-target"><ShieldAlert v-if="status==='changed'||status==='invalid'" :size="22"/><ShieldCheck v-else :size="22"/><div><strong>{{connection.name}}</strong><p>{{connection.host}}:{{connection.port}}</p></div><span class="certificate-badge">{{labels[status]}}</span></div>
        <p v-if="status==='changed'" class="certificate-warning" role="alert">这台电脑的证书或连接地址与之前不同。请核对新旧信息后，再决定是否更新信任。更新信任将断开该连接的已有会话。</p>
        <p v-else-if="status==='invalid'" class="certificate-warning" role="alert">证书已过期或尚未生效，请先更新远端电脑的证书。</p>
        <p v-else-if="inspection&& !ready" class="certificate-hint">首次连接，请确认这是你要连接的电脑。保存后仅信任当前用户这条连接对应的目标和证书，证书变化时会再次提示。</p>
        <p v-if="busy" role="status">正在处理证书…</p>
        <dl v-if="info" class="certificate-details">
          <dt>证书主体</dt><dd>{{info.subject||'未提供'}}</dd>
          <dt>签发者</dt><dd>{{info.issuer||'未提供'}}</dd>
          <dt>有效期</dt><dd>{{date(info.notBefore)}} 至 {{date(info.notAfter)}}</dd>
          <dt>校验结果</dt><dd>{{verification[info.verification]||'需要确认远端身份'}}</dd>
          <template v-if="status==='changed' && trust"><dt>之前的目标</dt><dd>{{trust.host}}:{{trust.port}}</dd><dt>之前的 SHA-256</dt><dd class="certificate-fingerprint">{{trust.certificate.fingerprint}}</dd></template>
          <dt>{{status==='changed'?'当前 SHA-256':'SHA-256 指纹'}}</dt><dd class="certificate-fingerprint">{{info.fingerprint}}</dd>
          <template v-if="trust"><dt>信任保存时间</dt><dd>{{date(trust.trustedAt)}}</dd></template>
        </dl>
        <p v-if="!info&&!busy&&!error">还没有保存此连接的证书信任。</p>
        <div v-if="revokePending" class="certificate-warning" role="alert">撤销后将断开此连接的已有会话，并取消待连接授权。下次连接需要重新检查证书。</div>
        <p v-if="notice" class="certificate-notice" role="status">{{notice}}</p>
        <p v-if="error" class="error" role="alert">{{error}}</p>
      </div>
      <div class="dialog-footer certificate-actions">
        <template v-if="revokePending">
          <span class="spacer"/><button type="button" :disabled="busy" @click="revokePending=false">取消撤销</button><button type="button" class="danger" :disabled="busy" @click="revoke">确认撤销</button>
        </template>
        <template v-else>
          <button v-if="trust" type="button" class="danger" :disabled="busy" @click="revokePending=true">撤销信任</button>
          <button v-else type="button" :disabled="busy||connection.protocol!=='rdp'" @click="inspect"><RefreshCw :size="14"/>重新检查</button>
          <span class="spacer"/><button type="button" data-cancel :disabled="busy" @click="close">{{forConnection?'取消':'关闭'}}</button>
          <button v-if="inspection && !ready" class="primary" :disabled="busy||!inspection.certificate.canTrust">{{status==='changed'?(forConnection?'更新信任并连接':'更新信任'):(forConnection?'信任此电脑并连接':'保存信任')}}</button>
          <button v-else-if="ready&&forConnection" type="button" class="primary" :disabled="busy" @click="emit('ready',inspection!.check)">继续连接</button>
          <button v-else-if="!inspection && trust" type="button" :disabled="busy||connection.protocol!=='rdp'" @click="inspect">重新检查</button>
        </template>
      </div>
    </form>
  </dialog>
</template>
<style>
.certificate-dialog{width:600px;}
.certificate-body{display:flex;flex-direction:column;gap:16px;}
.certificate-target{display:flex;align-items:center;gap:12px;min-width:0;}
.certificate-target>svg{flex-shrink:0;color:var(--blue);}
.certificate-target>div{min-width:0;flex:1;overflow-wrap:anywhere;}
.certificate-target strong{font-size:14px;}
.certificate-target p{font-size:12px;}
.certificate-badge{font-size:12px;color:var(--muted);white-space:nowrap;}
.certificate-hint,.certificate-warning{padding:12px;border-radius:7px;background:var(--surface);font-size:12px;line-height:1.7;}
.certificate-warning{color:var(--danger);border:1px solid var(--line);}
.certificate-details{display:grid;grid-template-columns:105px minmax(0,1fr);gap:12px 16px;margin:0;font-size:12px;line-height:1.6;}
.certificate-details dt{color:var(--muted);}
.certificate-details dd{margin:0;overflow-wrap:anywhere;user-select:text;}
.certificate-fingerprint{font-family:ui-monospace,SFMono-Regular,Consolas,monospace;}
.certificate-notice{color:var(--green);font-size:12px;}
.certificate-actions{flex-wrap:wrap;}
@media(max-width:600px){.certificate-details{grid-template-columns:1fr;gap:4px;}.certificate-details dd{margin-bottom:8px;}.certificate-badge{white-space:normal;max-width:86px;text-align:right;}.certificate-actions{gap:6px;}.certificate-actions .spacer{display:none;}.certificate-actions>button:last-child{margin-left:auto;}}
</style>
