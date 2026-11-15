<template>
  <div class="schema-fields">
    <div v-for="field in fields" :key="field.key" class="schema-field" :class="{ expanded: ['object','array','map'].includes(field.type), 'full-row': field.type === 'argv' }">
      <template v-if="field.type === 'object'">
        <div class="field-heading"><strong>{{ field.label }}</strong><el-switch v-if="field.optional" :model-value="!!modelValue?.[field.key]" @change="value => setField(field, value ? structuredDefaults(field.fields) : undefined)" /></div>
        <SchemaFields v-if="!field.optional || modelValue?.[field.key]" :fields="field.fields" :model-value="modelValue?.[field.key] || {}" @update:model-value="setField(field,$event)" />
      </template>
      <template v-else-if="field.type === 'array'">
        <div class="field-heading"><strong>{{ field.label }}</strong><el-button link type="primary" @click="add(field)">添加</el-button></div>
        <div v-for="(item,index) in modelValue?.[field.key] || []" :key="index" class="array-item"><div class="field-heading"><span>{{ field.itemLabel || field.label }} {{ index+1 }}</span><el-button link type="danger" @click="remove(field,index)">移除</el-button></div><SchemaFields :fields="field.fields" :model-value="item" @update:model-value="updateItem(field,index,$event)" /></div>
        <p v-if="!modelValue?.[field.key]?.length" class="field-hint">未配置</p>
      </template>
      <template v-else-if="field.type === 'map'">
        <div class="field-heading"><strong>{{ field.label }}</strong><el-button link type="primary" @click="addKey(field)">添加键值</el-button></div>
        <div v-for="(value,key) in modelValue?.[field.key] || {}" :key="key" class="map-row"><el-input :model-value="key" aria-label="键" @change="rename(field,key,$event)" /><el-input :model-value="String(value)" aria-label="值" @update:model-value="mapValue(field,key,$event)" /><el-button link type="danger" @click="mapRemove(field,key)">删除</el-button></div>
      </template>
      <template v-else>
        <label class="field-label">{{ field.label }} <span v-if="field.required" class="required">*</span></label>
        <el-select v-if="field.type === 'select'" :model-value="modelValue?.[field.key] ?? field.default" clearable filterable allow-create @update:model-value="setField(field,$event || undefined)"><el-option v-for="option in field.options" :key="String(option.value ?? option)" :label="option.label ?? String(option)" :value="option.value ?? option" /></el-select>
        <el-switch v-else-if="field.type === 'boolean'" :model-value="modelValue?.[field.key] ?? field.default ?? false" @update:model-value="setField(field,$event)" />
        <el-input v-else-if="field.type === 'argv'" :model-value="argumentText(field)" :placeholder="field.placeholder" @update:model-value="commitArguments(field,$event)" @change="!argumentErrors[field.key] && delete argumentDrafts[field.key]" />
        <el-input-number v-else-if="field.type === 'number'" :model-value="modelValue?.[field.key]" :min="field.min ?? 0" :max="field.max" controls-position="right" @update:model-value="setField(field,$event ?? undefined)" />
        <el-input v-else-if="field.type === 'lines'" type="textarea" :rows="3" :model-value="(modelValue?.[field.key] || []).join('\n')" :placeholder="field.placeholder || '每行一个数组项，保留项内的空格'" @update:model-value="setField(field,$event ? $event.split('\n') : undefined)" />
        <el-input v-else-if="field.type === 'port'" :model-value="String(modelValue?.[field.key] ?? '')" :placeholder="field.placeholder" @update:model-value="update(field.key,/^\d+$/.test($event) ? Number($event) : $event || undefined)" />
        <el-input v-else :type="field.type === 'textarea' ? 'textarea' : 'text'" :rows="3" :disabled="field.disabled" :model-value="modelValue?.[field.key] ?? ''" :placeholder="field.placeholder" @update:model-value="setField(field,$event || undefined)" />
        <p v-if="argumentErrors[field.key]" role="alert" class="required">{{ argumentErrors[field.key] }}</p>
      </template>
      <p v-if="field.help" class="field-hint">{{ field.help }}</p>
    </div>
  </div>
</template>
<script setup>
import { getCurrentInstance, inject, onBeforeUnmount, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { formatShell, splitShell } from '../utils/workloadYaml'
const props=defineProps({modelValue:Object,fields:{type:Array,required:true}})
const emit=defineEmits(['update:modelValue'])
const argumentDrafts=ref({}),argumentErrors=ref({})
const fieldErrors=inject('workloadFieldErrors',null),instanceId=getCurrentInstance().uid
onBeforeUnmount(()=>{if(fieldErrors)for(const key of Object.keys(fieldErrors))if(key.startsWith(instanceId+':'))delete fieldErrors[key]})
function argumentText(field){return argumentDrafts.value[field.key] ?? formatShell(props.modelValue?.[field.key])}
function commitArguments(field,value){argumentDrafts.value[field.key]=value;const key=instanceId+':'+field.key;try{const argv=splitShell(value);update(field.key,argv.length?argv:undefined);delete argumentErrors.value[field.key];if(fieldErrors)delete fieldErrors[key]}catch(e){argumentErrors.value[field.key]=e.message;if(fieldErrors)fieldErrors[key]=field.label+'：'+e.message}}
const clone=value=>JSON.parse(JSON.stringify(value ?? {}))
function update(key,value){const result={...props.modelValue};if(value===undefined)delete result[key];else result[key]=value;emit('update:modelValue',result)}
function setField(field,value){const result={...props.modelValue};if(value===undefined)delete result[field.key];else{for(const key of field.exclude||[])delete result[key];result[field.key]=value}emit('update:modelValue',result)}
function structuredDefaults(fields){return Object.fromEntries(fields.filter(f=>f.default!==undefined).map(f=>[f.key,clone(f.default)]))}
function add(field){update(field.key,[...(props.modelValue?.[field.key]||[]),field.initial?clone(field.initial):structuredDefaults(field.fields)])}
function remove(field,index){update(field.key,(props.modelValue?.[field.key]||[]).filter((_,i)=>i!==index))}
function updateItem(field,index,value){const items=[...props.modelValue[field.key]];items[index]=value;update(field.key,items)}
function addKey(field){const entries={...props.modelValue?.[field.key]};let key='key',i=1;while(key in entries)key='key-'+i++;entries[key]='';update(field.key,entries)}
function rename(field,key,next){const entries={...props.modelValue[field.key]};if(!next || next!==key && next in entries){ElMessage.error('键不能为空或重复');return}const value=entries[key];delete entries[key];entries[next]=value;update(field.key,entries)}
function mapValue(field,key,value){update(field.key,{...props.modelValue[field.key],[key]:value})}
function mapRemove(field,key){const entries={...props.modelValue[field.key]};delete entries[key];update(field.key,entries)}
</script>
<style scoped>
.full-row{grid-column:1/-1}.full-row :deep(input){font-family:monospace}
.schema-fields{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:18px}.schema-field{min-width:0}.schema-field.expanded{grid-column:1/-1;border-top:1px solid var(--ks-border);padding-top:16px}.field-heading{display:flex;align-items:center;justify-content:space-between;gap:12px;margin-bottom:12px;font-size:13px}.field-label{display:block;font-size:12px;margin-bottom:8px}.field-hint{color:var(--ks-text-secondary);font-size:12px;line-height:1.7;margin:7px 0}.array-item{background:var(--ks-body-bg);padding:16px;border-radius:6px;margin-bottom:12px}.map-row{display:grid;grid-template-columns:1fr 1fr 50px;gap:10px;margin:8px 0}.required{color:var(--ks-danger)}.schema-field>.el-select,.schema-field>.el-input-number{width:100%}@media(max-width:700px){.schema-fields{grid-template-columns:1fr}}
</style>
