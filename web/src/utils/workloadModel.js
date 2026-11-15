const CreateModel = (() => {
  const kinds = ['Deployment', 'StatefulSet', 'DaemonSet'];
  const clone = value => JSON.parse(JSON.stringify(value));
  const record = value => value !== null && typeof value === 'object' && !Array.isArray(value);
  const own = (value, key) => Object.prototype.hasOwnProperty.call(value, key);
  const present = value => value !== undefined && value !== null && value !== '';

  function initial(namespace = 'default') {
    return {
      apiVersion: 'apps/v1',
      kind: 'Deployment',
      metadata: { name: 'my-app', namespace: namespace || 'default', labels: { app: 'my-app' }, annotations: {} },
      spec: {
        replicas: 2,
        revisionHistoryLimit: 10,
        progressDeadlineSeconds: 600,
        strategy: { type: 'RollingUpdate', rollingUpdate: { maxSurge: '25%', maxUnavailable: '25%' } },
        selector: { matchLabels: { app: 'my-app' } },
        template: {
          metadata: { labels: { app: 'my-app' }, annotations: {} },
          spec: {
            restartPolicy: 'Always',
            dnsPolicy: 'ClusterFirst',
            terminationGracePeriodSeconds: 30,
            containers: [{
              name: 'my-app', image: 'nginx:1.27.5', imagePullPolicy: 'IfNotPresent',
              ports: [{ name: 'http', containerPort: 80, protocol: 'TCP' }],
              resources: { requests: { cpu: '100m', memory: '128Mi' }, limits: { cpu: '500m', memory: '512Mi' } },
              env: [], envFrom: [], volumeMounts: []
            }],
            initContainers: [], volumes: [], imagePullSecrets: []
          }
        }
      }
    };
  }

  function simplify(object) {
    const output = clone(object);
    const clean = resource => {
      if (!record(resource)) return;
      delete resource.status;
      if (record(resource.metadata)) {
        ['uid', 'resourceVersion', 'generation', 'creationTimestamp', 'managedFields', 'selfLink', 'deletionTimestamp'].forEach(key => delete resource.metadata[key]);
      }
      if (resource.kind === 'List' && Array.isArray(resource.items)) resource.items.forEach(clean);
    };
    if (Array.isArray(output)) output.forEach(clean);
    else clean(output);
    return output;
  }

  // Resource quantities use Kubernetes decimal, binary and exponent suffixes.
  function quantity(value) {
    if (!present(value)) return null;
    const match = String(value).trim().match(/^([+-]?(?:\d+(?:\.\d*)?|\.\d+))((?:[eE][+-]?\d+)|(?:[KMGTPE]i)|[numkKMGTPE])?$/);
    if (!match) return NaN;
    const suffix = match[2] || '';
    const factors = { n: 1e-9, u: 1e-6, m: 1e-3, k: 1e3, K: 1e3, M: 1e6, G: 1e9, T: 1e12, P: 1e15, E: 1e18, Ki: 1024, Mi: 1024 ** 2, Gi: 1024 ** 3, Ti: 1024 ** 4, Pi: 1024 ** 5, Ei: 1024 ** 6 };
    const factor = suffix ? (own(factors, suffix) ? factors[suffix] : 10 ** Number(suffix.slice(1))) : 1;
    return Number(match[1]) * factor;
  }

  function validate(object) {
    const errors = [];
    const add = (path, message) => errors.push({ path, message });
    if (!record(object)) { add('$', '资源必须是一个对象'); return errors; }
    if (object.apiVersion !== 'apps/v1') add('apiVersion', '工作负载的 apiVersion 必须为 apps/v1');
    if (!kinds.includes(object.kind)) add('kind', '请选择 Deployment、StatefulSet 或 DaemonSet');
    const metadata = record(object.metadata) ? object.metadata : {};
    const dnsLabel = /^[a-z0-9](?:[-a-z0-9]*[a-z0-9])?$/;
    const name = metadata.name;
    if (typeof name !== 'string' || name.length > 253 || !name.split('.').every(part => part.length <= 63 && dnsLabel.test(part))) add('metadata.name', '名称必填，使用小写字母、数字、连字符或点，最长 253 个字符');
    const namespace = metadata.namespace;
    if (typeof namespace !== 'string' || namespace.length > 63 || !dnsLabel.test(namespace)) add('metadata.namespace', '命名空间必填，使用小写字母、数字和连字符，最长 63 个字符');
    const spec = record(object.spec) ? object.spec : {};
    const template = record(spec.template) ? spec.template : {};
    const labels = record(template.metadata?.labels) ? template.metadata.labels : {};
    const selector = record(spec.selector) ? spec.selector : {};
    const matchLabels = record(selector.matchLabels) ? selector.matchLabels : {};
    const expressions = Array.isArray(selector.matchExpressions) ? selector.matchExpressions : [];
    if (!Object.keys(matchLabels).length && !expressions.length) add('spec.selector', '至少设置一个选择器，且必须匹配 Pod 模板标签');
    Object.entries(matchLabels).forEach(([key, value]) => {
      if (labels[key] !== value) add(`spec.selector.matchLabels.${key}`, `选择器 ${key} 必须与 Pod 模板标签一致`);
    });
    expressions.forEach((expression, index) => {
      const path = `spec.selector.matchExpressions[${index}]`;
      if (!record(expression)) { add(path, '标签表达式必须是一个对象'); return; }
      const { key, operator, values } = expression;
      if (typeof key !== 'string' || !key) { add(`${path}.key`, '表达式标签名必填'); return; }
      const exists = own(labels, key);
      if (!['In', 'NotIn', 'Exists', 'DoesNotExist'].includes(operator)) { add(`${path}.operator`, '无效的标签表达式操作符'); return; }
      if (['In', 'NotIn'].includes(operator) && (!Array.isArray(values) || !values.length)) { add(`${path}.values`, 'In / NotIn 至少需要一个值'); return; }
      if (['Exists', 'DoesNotExist'].includes(operator) && Array.isArray(values) && values.length) add(`${path}.values`, 'Exists / DoesNotExist 不能设置值');
      const matches = operator === 'In' ? exists && values.includes(labels[key]) : operator === 'NotIn' ? !exists || !values.includes(labels[key]) : operator === 'Exists' ? exists : !exists;
      if (!matches) add(path, `标签表达式 ${key} 不匹配 Pod 模板标签`);
    });
    if (object.kind === 'StatefulSet' && (typeof spec.serviceName !== 'string' || !spec.serviceName.trim())) add('spec.serviceName', 'StatefulSet 必须指定服务名称');
    if (object.kind === 'DaemonSet' && own(spec, 'replicas')) add('spec.replicas', 'DaemonSet 按节点运行，不能设置 replicas');
    if (object.kind !== 'DaemonSet' && present(spec.replicas) && (!Number.isInteger(spec.replicas) || spec.replicas < 0)) add('spec.replicas', '副本数必须是大于或等于 0 的整数');
    const podSpec = record(template.spec) ? template.spec : {};
    const containers = Array.isArray(podSpec.containers) ? podSpec.containers : [];
    const initContainers = Array.isArray(podSpec.initContainers) ? podSpec.initContainers : [];
    if (!containers.length) add('spec.template.spec.containers', '至少配置一个普通容器');
    const volumes = Array.isArray(podSpec.volumes) ? podSpec.volumes : [];
    const volumeNames = new Set();
    volumes.forEach((volume, index) => {
      const path = `spec.template.spec.volumes[${index}].name`;
      if (!record(volume) || typeof volume.name !== 'string' || !volume.name) { add(path, '存储卷名称必填'); return; }
      if (volumeNames.has(volume.name)) add(path, `存储卷名称 ${volume.name} 重复`);
      volumeNames.add(volume.name);
    });
    // StatefulSet claim templates are also valid volumeMount references.
    (Array.isArray(spec.volumeClaimTemplates) ? spec.volumeClaimTemplates : []).forEach((claim, index) => {
      const claimName = claim?.metadata?.name;
      const path = `spec.volumeClaimTemplates[${index}].metadata.name`;
      if (typeof claimName !== 'string' || !claimName) { add(path, '存储声明模板名称必填'); return; }
      if (volumeNames.has(claimName)) add(path, `存储声明模板与存储卷名称 ${claimName} 重复`);
      volumeNames.add(claimName);
    });
    const containerNames = new Set();
    [['containers', containers], ['initContainers', initContainers]].forEach(([group, items]) => items.forEach((container, index) => {
      const path = `spec.template.spec.${group}[${index}]`;
      if (!record(container)) { add(path, '容器必须是一个对象'); return; }
      if (typeof container.name !== 'string' || !container.name || container.name.length > 63 || !dnsLabel.test(container.name)) add(`${path}.name`, '容器名称必填，使用小写字母、数字和连字符，最长 63 个字符');
      else if (containerNames.has(container.name)) add(`${path}.name`, `容器名称 ${container.name} 重复，初始化容器也必须唯一`);
      containerNames.add(container.name);
      if (typeof container.image !== 'string' || !container.image.trim()) add(`${path}.image`, '容器镜像必填');
      ['cpu', 'memory'].forEach(resource => {
        const requests = container.resources?.requests;
        const limits = container.resources?.limits;
        const request = quantity(requests?.[resource]);
        const limit = quantity(limits?.[resource]);
        if (request !== null && (!Number.isFinite(request) || request < 0)) add(`${path}.resources.requests.${resource}`, '资源请求必须是有效的非负 Kubernetes 数量');
        if (limit !== null && (!Number.isFinite(limit) || limit < 0)) add(`${path}.resources.limits.${resource}`, '资源上限必须是有效的非负 Kubernetes 数量');
        if (Number.isFinite(request) && Number.isFinite(limit) && request > limit) add(`${path}.resources.requests.${resource}`, '资源请求不能超过资源上限');
      });
      const portNames = new Set();
      (Array.isArray(container.ports) ? container.ports : []).forEach((port, portIndex) => {
        const portPath = `${path}.ports[${portIndex}]`;
        if (!record(port)) { add(portPath, '端口必须是一个对象'); return; }
        ['containerPort', 'hostPort'].forEach(key => {
          if ((key === 'containerPort' || present(port[key])) && (!Number.isInteger(port[key]) || port[key] < 1 || port[key] > 65535)) add(`${portPath}.${key}`, '端口必须是 1–65535 的整数');
        });
        if (port.name) {
          if (portNames.has(port.name)) add(`${portPath}.name`, `端口名称 ${port.name} 重复`);
          portNames.add(port.name);
        }
      });
      ['readinessProbe', 'livenessProbe', 'startupProbe'].forEach(probeKey => {
        const probe = container[probeKey];
        if (probe === undefined) return;
        const probePath = `${path}.${probeKey}`;
        if (!record(probe)) { add(probePath, '探针必须是一个对象；关闭探针时移除此字段'); return; }
        const actions = ['httpGet', 'tcpSocket', 'exec', 'grpc'].filter(key => present(probe[key]));
        if (actions.length !== 1) add(probePath, '每个探针必须且只能选择一种检测方式');
        ['successThreshold', 'failureThreshold'].forEach(key => {
          if (present(probe[key]) && (!Number.isInteger(probe[key]) || probe[key] < 1)) add(`${probePath}.${key}`, '阈值必须是大于或等于 1 的整数');
        });
        if (probeKey !== 'readinessProbe' && present(probe.successThreshold) && probe.successThreshold !== 1) add(`${probePath}.successThreshold`, '存活与启动探针的成功阈值必须为 1');
        ['initialDelaySeconds', 'terminationGracePeriodSeconds'].forEach(key => {
          if (present(probe[key]) && (!Number.isInteger(probe[key]) || probe[key] < (key === 'terminationGracePeriodSeconds' ? 1 : 0))) add(`${probePath}.${key}`, key === 'initialDelaySeconds' ? '初始延迟必须是非负整数' : '终止宽限时间必须是正整数');
        });
        ['periodSeconds', 'timeoutSeconds'].forEach(key => {
          if (present(probe[key]) && (!Number.isInteger(probe[key]) || probe[key] < 1)) add(`${probePath}.${key}`, '间隔与超时必须是正整数');
        });
        ['httpGet', 'tcpSocket', 'grpc'].forEach(actionKey => {
          const action = probe[actionKey];
          if (!record(action)) return;
          const port = action.port;
          const named = typeof port === 'string' && port.length <= 15 && /^[a-z0-9](?:[-a-z0-9]*[a-z0-9])?$/.test(port) && /[a-z]/.test(port);
          if (!(Number.isInteger(port) && port >= 1 && port <= 65535) && !(actionKey !== 'grpc' && named)) add(`${probePath}.${actionKey}.port`, '探针端口必须为 1–65535，HTTP/TCP 也可使用合法端口名');
        });
        if (record(probe.exec) && (!Array.isArray(probe.exec.command) || !probe.exec.command.length || probe.exec.command.some(value => typeof value !== 'string'))) add(`${probePath}.exec.command`, 'Exec 探针至少需要一个命令参数');
      });
      const mountPaths = new Set();
      (Array.isArray(container.volumeMounts) ? container.volumeMounts : []).forEach((mount, mountIndex) => {
        const mountPath = `${path}.volumeMounts[${mountIndex}]`;
        if (!record(mount)) { add(mountPath, '挂载必须是一个对象'); return; }
        if (!volumeNames.has(mount.name)) add(`${mountPath}.name`, `找不到存储卷或声明模板 ${mount.name || '（未选择）'}`);
        if (typeof mount.mountPath !== 'string' || !mount.mountPath.startsWith('/')) add(`${mountPath}.mountPath`, '挂载路径必须是绝对路径');
        else if (mountPaths.has(mount.mountPath)) add(`${mountPath}.mountPath`, `挂载路径 ${mount.mountPath} 重复`);
        mountPaths.add(mount.mountPath);
      });
    }));
    return errors;
  }

  function setKind(object, kind) {
    if (!kinds.includes(kind)) throw new Error(`不支持的工作负载类型：${kind}`);
    const output = clone(object);
    if (output.kind === kind) return output;
    output.apiVersion = 'apps/v1';
    output.kind = kind;
    if (!record(output.spec)) output.spec = {};
    const spec = output.spec;
    // Remove only known fields that cannot belong to the target kind.
    if (kind !== 'Deployment') {
      delete spec.strategy;
      delete spec.progressDeadlineSeconds;
      delete spec.paused;
    }
    if (kind !== 'StatefulSet') {
      ['serviceName', 'podManagementPolicy', 'volumeClaimTemplates', 'persistentVolumeClaimRetentionPolicy', 'ordinals'].forEach(key => delete spec[key]);
    }
    if (kind === 'Deployment') {
      delete spec.updateStrategy;
      spec.replicas = Number.isInteger(spec.replicas) ? spec.replicas : 2;
      spec.strategy = { type: 'RollingUpdate', rollingUpdate: { maxSurge: '25%', maxUnavailable: '25%' } };
      spec.progressDeadlineSeconds = 600;
    } else if (kind === 'StatefulSet') {
      spec.replicas = Number.isInteger(spec.replicas) ? spec.replicas : 2;
      spec.serviceName = output.metadata?.name || 'my-app';
      spec.podManagementPolicy = 'OrderedReady';
      spec.updateStrategy = { type: 'RollingUpdate', rollingUpdate: { partition: 0 } };
    } else {
      delete spec.replicas;
      spec.updateStrategy = { type: 'RollingUpdate', rollingUpdate: { maxUnavailable: 1 } };
    }
    return output;
  }
  return { initial, simplify, validate, setKind };
})();

export const { initial, simplify, setKind } = CreateModel
export function validate(object) {
  const errors=CreateModel.validate(object)
  const pod=object?.spec?.template?.spec || {}
  for(const key of ['containers','initContainers','volumes','imagePullSecrets','tolerations'])if(pod[key]!==undefined&&!Array.isArray(pod[key]))errors.push({path:`spec.template.spec.${key}`,message:'必须是数组'})
  for(const [i,v] of (Array.isArray(pod.volumes)?pod.volumes:[]).entries()) if(Object.keys(v||{}).filter(key=>key!=='name'&&v[key]!==undefined).length!==1) errors.push({path:`volumes[${i}]`,message:'一个卷必须且只能选择一种来源'})
  for(const container of [...(Array.isArray(pod.containers)?pod.containers:[]),...(Array.isArray(pod.initContainers)?pod.initContainers:[])]) {
    if(!container || typeof container!=='object')continue
    for(const key of ['command','args'])if(container[key]!==undefined&&(!Array.isArray(container[key])||container[key].some(value=>typeof value!=='string')))errors.push({path:`${container.name}.${key}`,message:'命令和参数必须是字符串数组'})
    for(const env of Array.isArray(container.env)?container.env:[]) { if(env?.valueFrom && env.value !== undefined) errors.push({path:`${container.name}.env.${env.name}`,message:'直接值与引用来源不能同时配置'}); if(env?.valueFrom && Object.keys(env.valueFrom).length!==1)errors.push({path:`${container.name}.env.${env.name}`,message:'必须且只能配置一个环境变量引用来源'}) }
    for(const env of Array.isArray(container.envFrom)?container.envFrom:[])if(['configMapRef','secretRef'].filter(key=>env?.[key]!==undefined).length!==1)errors.push({path:`${container.name}.envFrom`,message:'必须且只能选择一种整组引用来源'})
  }
  return errors
}
