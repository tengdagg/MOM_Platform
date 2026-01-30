<template>
  <span class="os-icon" :title="osName">
    <img v-if="iconPath" :src="iconPath" :width="size" :height="size" :alt="osName" />
    <el-icon v-else :size="size">
      <Platform />
    </el-icon>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Platform } from '@element-plus/icons-vue'

// 导入所有 OS 图标
import debianIcon from '@/assets/os-icons/debian.svg'
import ubuntuIcon from '@/assets/os-icons/ubuntu.svg'
import centosIcon from '@/assets/os-icons/centos.svg'
import redhatIcon from '@/assets/os-icons/redhat.svg'
import rockyIcon from '@/assets/os-icons/rocky.svg'
import almaIcon from '@/assets/os-icons/alma.svg'
import suseIcon from '@/assets/os-icons/suse.svg'
import fedoraIcon from '@/assets/os-icons/fedora.svg'
import archIcon from '@/assets/os-icons/arch.svg'
import alpineIcon from '@/assets/os-icons/alpine.svg'
import amazonIcon from '@/assets/os-icons/amazon.svg'
import oracleIcon from '@/assets/os-icons/oracle.svg'
import windowsIcon from '@/assets/os-icons/windows.svg'
import macosIcon from '@/assets/os-icons/macos.svg'
import freebsdIcon from '@/assets/os-icons/freebsd.svg'
import linuxIcon from '@/assets/os-icons/linux.svg'

const props = defineProps<{
  os: string
  size?: number
}>()

const osName = computed(() => props.os || 'Unknown')
const osLower = computed(() => (props.os || '').toLowerCase())

const iconPath = computed(() => {
  const os = osLower.value
  
  if (os.includes('debian')) return debianIcon
  if (os.includes('ubuntu')) return ubuntuIcon
  if (os.includes('centos')) return centosIcon
  if (os.includes('redhat') || os.includes('rhel') || os.includes('red hat')) return redhatIcon
  if (os.includes('rocky')) return rockyIcon
  if (os.includes('alma')) return almaIcon
  if (os.includes('suse') || os.includes('sles')) return suseIcon
  if (os.includes('fedora')) return fedoraIcon
  if (os.includes('arch')) return archIcon
  if (os.includes('alpine')) return alpineIcon
  if (os.includes('amazon') || os.includes('amzn')) return amazonIcon
  if (os.includes('oracle')) return oracleIcon
  if (os.includes('windows') || os.includes('win')) return windowsIcon
  if (os.includes('macos') || os.includes('darwin') || os.includes('mac os')) return macosIcon
  if (os.includes('freebsd') || os.includes('bsd')) return freebsdIcon
  if (os.includes('linux') || os.includes('gnu')) return linuxIcon
  
  return null
})
</script>

<style scoped>
.os-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.os-icon img {
  flex-shrink: 0;
  object-fit: contain;
}
</style>
