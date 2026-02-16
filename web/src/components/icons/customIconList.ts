// 自定义图标名称列表（与 CustomIcons.vue 中定义的图标一一对应）
// Layout.vue 和 Menus.vue 共用此列表，确保图标解析一致
export const customIconNames = [
    'Helm',
    'Kubernetes',
    'Aliyun',
    'Tencent',
    'AWS',
    'JDCloud',
    'Baidu',
    'KSYun',
    'skills',
    'Aibot',
    'NetworkDevice',
    'Audit',
    'Diagnosis'
]

// 判断是否为自定义图标
export const isCustomIcon = (name: string): boolean => {
    return customIconNames.includes(name)
}
