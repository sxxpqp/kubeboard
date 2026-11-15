export async function copyText(value) {
  if (navigator.clipboard?.writeText) {
    try {await navigator.clipboard.writeText(value);return} catch { /* HTTP deployments may require the selection API. */ }
  }
  const previous=document.activeElement
  const textarea=document.createElement('textarea')
  textarea.value=value
  textarea.setAttribute('readonly','')
  textarea.style.cssText='position:fixed;left:-9999px;top:0'
  document.body.appendChild(textarea)
  try {
    textarea.select()
    if(!document.execCommand('copy'))throw new Error('请选中 YAML 并手动复制')
  } finally {textarea.remove();previous?.focus()}
}
