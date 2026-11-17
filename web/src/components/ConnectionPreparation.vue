<script setup lang="ts">
import { ref, watch } from "vue";
import BaseModal from "./BaseModal.vue";

const props = defineProps<{ open: boolean }>();
const system = ref("windows");
watch(() => props.open, (open) => { if (open) system.value = "windows"; });
const emit = defineEmits<{ close: [] }>();
</script>

<template>
  <BaseModal
    :open="open"
    title="连接前准备"
    description="选择被连接电脑的系统。先确认开机时能够连接，再测试网络唤醒。"
    size="lg"
    @close="emit('close')"
  >
    <el-tabs v-model="system" class="preparation-tabs" aria-label="被连接电脑的系统">
      <el-tab-pane label="Windows" name="windows">
        <div v-if="system === 'windows'" role="region" aria-label="Windows 连接准备说明" tabindex="0" class="preparation-guide space-y-5 text-[13px] leading-6 text-muted">
          <section aria-labelledby="prepare-windows-rdp">
            <h3 id="prepare-windows-rdp" class="mb-2 text-sm font-bold text-ink">开启 RDP 远程桌面</h3>
            <ol class="list-decimal space-y-2 pl-5">
              <li>在目标电脑的“设置 → 系统 → 系统信息（关于）”查看 Windows 版本。专业版、企业版、教育版及 Windows Server 支持作为远程桌面主机；家庭版不能接受系统自带的 RDP 连接。</li>
              <li>打开“设置 → 系统 → 远程桌面”，启用远程桌面。管理员账户默认可连接；其他账户需添加到“远程桌面用户”。保留网络级别身份验证 (NLA)。</li>
              <li>准备目标电脑的账户名和账户密码。使用 Windows 账户密码，不是 Windows Hello PIN；本地账户可用“电脑名\用户名”，域账户可用“域名\用户名”。Microsoft 账户使用对应的登录邮箱和密码。</li>
              <li>在命令提示符运行 <code>ipconfig</code>，找到当前连接网卡的 IPv4 地址。在本应用新建连接，协议选 RDP，主机地址填该 IP，端口默认 3389；若已改过远程桌面端口，填写实际端口。</li>
              <li>选择或新建 RDP 凭据，填写上述账户。让目标电脑保持开机且暂不睡眠，先检查 NAS 到目标端口的 TCP 连通性，再点击“连接”。确认 Windows 防火墙允许当前网络配置文件下的远程桌面入站连接，无需关闭整个防火墙。</li>
            </ol>
            <a class="mt-2 inline-block text-ink underline underline-offset-4" href="https://learn.microsoft.com/zh-cn/windows-server/remote/remote-desktop-services/remotepc/remote-desktop-allow-access" target="_blank" rel="noopener noreferrer">Microsoft：启用远程桌面 ↗</a>
          </section>
          <section aria-labelledby="prepare-windows-wol" class="border-t border-line pt-4">
            <h3 id="prepare-windows-wol" class="mb-2 text-sm font-bold text-ink">设置网络唤醒 (WOL)</h3>
            <ol class="list-decimal space-y-2 pl-5">
              <li>确认主板和网卡支持 WOL，接通电源并优先使用有线以太网。在 BIOS / UEFI 中启用 Wake on LAN 或 PCI-E 唤醒；具体名称与支持的电源状态以电脑或主板说明书为准。</li>
              <li>在“设备管理器 → 网络适配器”打开实际联网的以太网卡属性。在“高级”中启用 Wake on Magic Packet（魔术封包唤醒）；在“电源管理”中勾选“允许此设备唤醒计算机”和“只允许幻数据包唤醒计算机”。不同网卡、驱动及电源模式可能不提供这些选项，需查阅厂商说明。</li>
              <li>运行 <code>ipconfig /all</code>，记录该以太网卡的“物理地址”（MAC 地址）。不要填 Wi-Fi、蓝牙或虚拟网卡的地址。</li>
              <li>编辑本应用中的 Windows 连接，启用“网络唤醒 (WOL)”并填写该 MAC 地址。广播地址先留空，由 NAS 自动选择发送网络；需要手动填写时，使用目标网卡所在、且 NAS 直连子网的广播地址。标准局域网广播通常不能跨路由或 VLAN。</li>
              <li>先在电脑开机时确认 RDP 能正常连接，再让电脑进入“睡眠”，从本应用的唤醒入口发送唤醒包。稍等后重新检查端口并连接；若睡眠可唤醒，再根据厂商支持情况测试休眠或关机。</li>
            </ol>
            <p class="mt-3 rounded-lg border border-line bg-soft p-3">“唤醒包已发送”只表示 NAS 已发送数据包，不代表电脑已唤醒。关机和快速启动状态能否唤醒取决于固件、网卡、驱动及供电；睡眠成功不代表关机也能成功。Wi-Fi、USB 网卡、扩展坞和笔记本电池模式需要分别实测。</p>
            <div class="mt-2 flex flex-wrap gap-x-4 gap-y-1">
              <a class="text-ink underline underline-offset-4" href="https://learn.microsoft.com/zh-cn/troubleshoot/windows-client/setup-upgrade-and-drivers/wake-on-lan-feature" target="_blank" rel="noopener noreferrer">Microsoft：WOL 与电源状态 ↗</a>
              <a class="text-ink underline underline-offset-4" href="https://www.dell.com/support/kbdoc/en-us/000129137/wake-on-lan-wol-troubleshooting-best-practices" target="_blank" rel="noopener noreferrer">Dell：BIOS 与网卡设置参考 ↗</a>
            </div>
          </section>
          <section aria-labelledby="prepare-windows-troubleshooting" class="border-t border-line pt-4">
            <h3 id="prepare-windows-troubleshooting" class="mb-2 text-sm font-bold text-ink">连接或唤醒失败时</h3>
            <ul class="list-disc space-y-2 pl-5">
              <li>端口不可达：检查电脑是否开机、IP 和端口是否正确、远程桌面是否启用，以及 NAS 到电脑的网络和防火墙规则。端口可达只表示 TCP 连接成功，不代表账户认证成功。</li>
              <li>账户或密码被拒绝：确认账户有远程桌面权限，使用账户密码而非 PIN，并核对电脑名或域名。不要通过关闭 NLA 来代替修正账户配置。</li>
              <li>睡眠后无法唤醒：核对物理网卡 MAC、BIOS 和驱动设置、供电及广播网络；没有相关选项时查阅设备厂商说明。若仅关机后失败，重点检查厂商对关机唤醒和快速启动的支持。</li>
              <li>唤醒后仍不能连接：等待系统启动完成，再检查 IP 是否变化及远程桌面端口是否恢复。可在路由器中为电脑保留固定的 DHCP 地址。</li>
            </ul>
            <p class="mt-2">远程桌面端口仅供可信网络访问，不要直接映射到公网；跨网络连接应使用可信的 VPN 等接入方式。</p>
          </section>
        </div>
      </el-tab-pane>
      <el-tab-pane label="Mac" name="mac">
        <div v-if="system === 'mac'" role="region" aria-label="Mac 连接准备说明" tabindex="0" class="preparation-guide space-y-5 text-[13px] leading-6 text-muted">
      <section aria-labelledby="prepare-mac-vnc">
        <h3 id="prepare-mac-vnc" class="mb-2 text-sm font-bold text-ink">Mac：开启 VNC 屏幕共享</h3>
        <ol class="list-decimal space-y-2 pl-5">
          <li>在 Mac 打开“系统设置 → 通用 → 共享”。如果已开启“远程管理”，先关闭它，再开启“屏幕共享”。</li>
          <li>点击“屏幕共享”旁的详情按钮，开启“VNC 显示程序可以使用密码控制屏幕”，并设置独立的 VNC 密码。应用使用这个密码连接，不是 Mac 登录密码。</li>
          <li>在“系统设置 → 网络 → 当前连接 → 详细信息”查看 Mac 的局域网 IP。</li>
          <li>在本应用新建连接，协议选 VNC，主机地址填 Mac 的局域网 IP，端口填 5900。选择或新建 VNC 凭据，填入刚设置的 VNC 密码。</li>
          <li>保持 Mac 清醒，先检测 TCP 连通性，再点击“连接”。NAS 必须能够访问 Mac 的 5900 端口。</li>
        </ol>
        <a class="mt-2 inline-block text-ink underline underline-offset-4" href="https://support.apple.com/zh-cn/guide/mac-help/mh11848/mac" target="_blank" rel="noopener noreferrer">Apple：开启屏幕共享 ↗</a>
      </section>

      <section aria-labelledby="prepare-mac-wol" class="border-t border-line pt-4">
        <h3 id="prepare-mac-wol" class="mb-2 text-sm font-bold text-ink">Mac：设置网络唤醒</h3>
        <ol class="list-decimal space-y-2 pl-5">
          <li>MacBook：打开“系统设置 → 电池 → 选项”，将“唤醒以供网络访问”设为“始终”或“仅使用电源适配器时”。</li>
          <li>Mac mini、iMac 等台式机：打开“系统设置 → 能源／节能”，开启“唤醒以供网络访问”。名称和可用选项因系统及机型而异。</li>
          <li>首次测试建议接通电源，优先使用有线网络；MacBook 保持开盖。在“系统设置 → 网络 → 以太网 → 详细信息 → 硬件”查看该网卡的 MAC 地址。</li>
          <li>编辑本应用中的 VNC 连接，启用“网络唤醒 (WOL)”，填写对应网卡的 MAC 地址。广播地址先留空，由 NAS 自动选择发送网络；无法确定时再填写 NAS 直连子网的广播地址。</li>
          <li>保存后让 Mac 进入“睡眠”，从该连接的唤醒入口发送唤醒包，稍等后再连接。NAS 与 Mac 需位于同一广播域。</li>
        </ol>
        <p class="mt-3 rounded-lg border border-line bg-soft p-3">“唤醒包已发送”不代表 Mac 已开机。标准 WOL 是否有效取决于机型、供电和网卡；Wi-Fi、外接网卡及合盖状态需要实测，完全关机后的远程开机不作保证。</p>
        <a class="mt-2 inline-block text-ink underline underline-offset-4" href="https://support.apple.com/zh-cn/guide/mac-help/mchle41a6ccd/mac" target="_blank" rel="noopener noreferrer">Apple：睡眠与唤醒设置 ↗</a>
      </section>

      <section aria-labelledby="prepare-mac-troubleshooting" class="border-t border-line pt-4">
        <h3 id="prepare-mac-troubleshooting" class="mb-2 text-sm font-bold text-ink">连接失败时</h3>
        <p>先确认 Mac 清醒、地址正确，且已启用 VNC 密码访问。如果端口不可达，检查 NAS 到 Mac 的网络及防火墙；如果密码被拒绝，检查凭据中填写的是否为 VNC 密码。</p>
        <p class="mt-2">VNC 端口仅供可信内网访问，不要直接暴露到公网。跨网络连接应使用可信的 VPN 等接入方式。</p>
      </section>
        </div>
      </el-tab-pane>
    </el-tabs>
    <template #footer>
      <el-button type="primary" @click="emit('close')">知道了</el-button>
    </template>
  </BaseModal>
</template>

<style>
.preparation-tabs .el-tabs__header { margin-bottom: 12px; }
.preparation-guide {
  max-height: max(140px, calc(100dvh - 285px));
  overflow-y: auto;
  overscroll-behavior: contain;
  padding-right: 10px;
  overflow-wrap: anywhere;
}
.preparation-guide code {
  border-radius: 4px;
  background: var(--surface-2);
  padding: 1px 4px;
  font-size: 12px;
}
</style>
