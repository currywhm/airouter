<template>
  <div class="min-h-screen bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white">
    <header class="sticky top-0 z-30 border-b border-gray-200/80 bg-white/95 backdrop-blur dark:border-dark-800 dark:bg-dark-950/95">
      <div class="mx-auto flex h-16 max-w-7xl items-center justify-between gap-4 px-4 sm:px-6">
        <RouterLink to="/login" class="flex min-w-0 items-center gap-3">
          <span class="flex h-9 w-9 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-white ring-1 ring-gray-200 dark:bg-dark-800 dark:ring-dark-700">
            <img :src="siteLogo || '/logo.svg'" :alt="siteName" class="h-full w-full object-contain" />
          </span>
          <span class="truncate text-sm font-semibold text-gray-950 dark:text-white">{{ siteName }}</span>
        </RouterLink>

        <div class="flex items-center gap-2">
          <LocaleSwitcher />
        </div>
      </div>
    </header>

    <div class="mx-auto grid max-w-7xl grid-cols-1 gap-8 px-4 py-8 sm:px-6 lg:grid-cols-[220px_minmax(0,1fr)] lg:gap-12 lg:py-10">
      <aside class="lg:sticky lg:top-24 lg:self-start">
        <div class="mb-4 flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-primary-600 dark:text-primary-400">
          <Icon name="book" size="sm" />
          {{ copy.docs }}
        </div>
        <nav class="flex gap-2 overflow-x-auto pb-1 lg:block lg:space-y-1 lg:overflow-visible" :aria-label="copy.contents">
          <a
            v-for="section in sections"
            :key="section.id"
            :href="`#${section.id}`"
            class="block shrink-0 rounded-md px-3 py-2 text-sm text-gray-600 transition hover:bg-gray-100 hover:text-gray-950 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white"
          >
            {{ section.title }}
          </a>
        </nav>
      </aside>

      <main class="min-w-0">
        <div class="mb-10 border-b border-gray-200 pb-8 dark:border-dark-800">
          <p class="mb-3 text-sm font-medium text-primary-600 dark:text-primary-400">{{ copy.eyebrow }}</p>
          <h1 class="text-3xl font-bold tracking-tight text-gray-950 dark:text-white sm:text-4xl">{{ copy.title }}</h1>
          <p class="mt-4 max-w-3xl text-base leading-7 text-gray-600 dark:text-dark-300">{{ copy.intro }}</p>
          <div class="mt-6 flex flex-wrap items-center gap-3 text-sm">
            <a href="#quick-start" class="inline-flex items-center gap-2 rounded-lg bg-primary-600 px-4 py-2 font-medium text-white transition hover:bg-primary-700">
              <Icon name="chevronRight" size="sm" />
              {{ copy.startNow }}
            </a>
            <span class="text-gray-500 dark:text-dark-400">{{ copy.updated }}</span>
          </div>
        </div>

        <section id="quick-start" class="scroll-mt-24">
          <SectionHeading :number="'01'" :title="copy.quickStartTitle" :description="copy.quickStartDescription" />
          <div class="mt-6 grid gap-4 md:grid-cols-2">
            <StepCard :number="'1'" :title="copy.steps.register.title" :body="copy.steps.register.body" />
            <StepCard :number="'2'" :title="copy.steps.wallet.title" :body="copy.steps.wallet.body" />
            <StepCard :number="'3'" :title="copy.steps.key.title" :body="copy.steps.key.body" />
            <StepCard :number="'4'" :title="copy.steps.client.title" :body="copy.steps.client.body" />
          </div>
          <div class="mt-4 rounded-lg border border-primary-200 bg-primary-50 p-4 text-sm leading-6 text-primary-900 dark:border-primary-500/30 dark:bg-primary-500/10 dark:text-primary-100">
            <strong>{{ copy.tipLabel }}</strong>{{ copy.quickTip }}
          </div>
        </section>

        <section id="endpoint" class="mt-14 scroll-mt-24">
          <SectionHeading :number="'02'" :title="copy.endpointTitle" :description="copy.endpointDescription" />
          <div class="mt-6 grid gap-4 md:grid-cols-2">
            <InfoPanel :title="copy.baseUrlLabel" :value="baseUrl" :copy-label="copy.copy" :copied="copiedKey === 'baseUrl'" @copy="copyValue(baseUrl, 'baseUrl')" />
            <InfoPanel :title="copy.responsesLabel" :value="`${baseUrl}/responses`" :copy-label="copy.copy" :copied="copiedKey === 'responsesUrl'" @copy="copyValue(`${baseUrl}/responses`, 'responsesUrl')" />
          </div>
          <div class="mt-4 overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-800 dark:bg-dark-900">
            <div class="border-b border-gray-200 px-4 py-3 text-sm font-semibold dark:border-dark-800">{{ copy.endpointTableTitle }}</div>
            <div class="divide-y divide-gray-100 text-sm dark:divide-dark-800">
              <div v-for="row in endpointRows" :key="row.name" class="grid gap-1 px-4 py-3 sm:grid-cols-[180px_1fr] sm:gap-4">
                <span class="text-gray-500 dark:text-dark-400">{{ row.name }}</span>
                <code class="break-all text-gray-800 dark:text-dark-200">{{ row.value }}</code>
              </div>
            </div>
          </div>
        </section>

        <section id="wallet" class="mt-14 scroll-mt-24">
          <SectionHeading :number="'03'" :title="copy.walletTitle" :description="copy.walletDescription" />
          <div class="mt-6 grid gap-3 sm:grid-cols-3">
            <div v-for="item in walletPoints" :key="item.title" class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-800 dark:bg-dark-900">
              <div class="mb-3 flex h-8 w-8 items-center justify-center rounded-md bg-gray-100 text-sm font-semibold text-gray-700 dark:bg-dark-800 dark:text-dark-200">{{ item.number }}</div>
              <h3 class="text-sm font-semibold text-gray-950 dark:text-white">{{ item.title }}</h3>
              <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ item.body }}</p>
            </div>
          </div>
        </section>

        <section id="models" class="mt-14 scroll-mt-24">
          <SectionHeading :number="'04'" :title="copy.modelsTitle" :description="copy.modelsDescription" />
          <div class="mt-6 overflow-x-auto rounded-lg border border-gray-200 bg-white dark:border-dark-800 dark:bg-dark-900">
            <table class="w-full min-w-[620px] text-left text-sm">
              <thead class="border-b border-gray-200 bg-gray-50 text-xs uppercase tracking-wide text-gray-500 dark:border-dark-800 dark:bg-dark-900/70 dark:text-dark-400">
                <tr>
                  <th class="px-4 py-3 font-medium">{{ copy.model }}</th>
                  <th class="px-4 py-3 font-medium">{{ copy.positioning }}</th>
                  <th class="px-4 py-3 font-medium">{{ copy.inputPrice }}</th>
                  <th class="px-4 py-3 font-medium">{{ copy.outputPrice }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-800">
                <tr v-for="model in models" :key="model.name">
                  <td class="px-4 py-3 font-medium text-gray-900 dark:text-white">{{ model.name }}</td>
                  <td class="px-4 py-3 text-gray-600 dark:text-dark-300">{{ model.positioning }}</td>
                  <td class="px-4 py-3 font-mono text-gray-700 dark:text-dark-200">{{ model.input }}</td>
                  <td class="px-4 py-3 font-mono text-gray-700 dark:text-dark-200">{{ model.output }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <p class="mt-3 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ copy.priceNote }}</p>
        </section>

        <section id="examples" class="mt-14 scroll-mt-24">
          <SectionHeading :number="'05'" :title="copy.examplesTitle" :description="copy.examplesDescription" />
          <div class="mt-6 space-y-4">
            <CodePanel :title="copy.curlTitle" :code="curlExample" :copy-label="copy.copy" :copied="copiedKey === 'curl'" @copy="copyValue(curlExample, 'curl')" />
            <CodePanel :title="copy.pythonTitle" :code="pythonExample" :copy-label="copy.copy" :copied="copiedKey === 'python'" @copy="copyValue(pythonExample, 'python')" />
          </div>
        </section>

        <section id="troubleshooting" class="mt-14 scroll-mt-24">
          <SectionHeading :number="'06'" :title="copy.troubleshootingTitle" :description="copy.troubleshootingDescription" />
          <div class="mt-6 divide-y divide-gray-200 rounded-lg border border-gray-200 bg-white dark:divide-dark-800 dark:border-dark-800 dark:bg-dark-900">
            <div v-for="item in troubleshooting" :key="item.title" class="px-4 py-4 sm:px-5">
              <h3 class="text-sm font-semibold text-gray-950 dark:text-white">{{ item.title }}</h3>
              <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ item.body }}</p>
            </div>
          </div>
        </section>

        <section id="support" class="mt-14 scroll-mt-24">
          <SectionHeading :number="'07'" :title="copy.supportTitle" :description="copy.supportDescription" />
          <div class="mt-6 grid gap-4 sm:grid-cols-2">
            <InfoPanel :title="copy.qqGroupLabel" :value="copy.qqGroupValue" :copy-label="copy.copy" :copied="copiedKey === 'qqGroup'" @copy="copyValue(copy.qqGroupValue, 'qqGroup')" />
            <div class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-800 dark:bg-dark-900">
              <div class="flex items-center justify-between gap-3">
                <span class="text-sm font-medium text-gray-600 dark:text-dark-300">{{ copy.qqLabel }}</span>
                <a
                  href="https://wpa.qq.com/msgrd?v=3&uin=3935287835&site=qq&menu=yes"
                  target="_blank"
                  rel="noreferrer"
                  class="inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium text-primary-700 hover:bg-primary-50 dark:text-primary-300 dark:hover:bg-primary-500/10"
                >
                  <Icon name="externalLink" size="xs" />
                  {{ copy.contactNow }}
                </a>
              </div>
              <div class="mt-3 flex items-center justify-between gap-3 rounded-md bg-gray-50 px-3 py-2 dark:bg-dark-800">
                <code class="break-all text-xs text-gray-800 dark:text-dark-200">3935287835</code>
                <button
                  type="button"
                  class="inline-flex shrink-0 items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium text-primary-700 hover:bg-primary-100 dark:text-primary-300 dark:hover:bg-primary-500/20"
                  @click="copyValue('3935287835', 'qq')"
                >
                  <Icon :name="copiedKey === 'qq' ? 'check' : 'copy'" size="xs" />
                  {{ copiedKey === 'qq' ? copy.copied : copy.copy }}
                </button>
              </div>
            </div>
          </div>
        </section>

        <footer class="mt-16 border-t border-gray-200 pt-6 text-sm text-gray-500 dark:border-dark-800 dark:text-dark-400">
          {{ copy.footer }}
        </footer>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const { locale } = useI18n()
const appStore = useAppStore()
const copiedKey = ref('')

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'AIRouter')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const baseUrl = 'http://100.90.218.124:18080/v1'
const isZh = computed(() => locale.value.startsWith('zh'))

const copy = computed(() => isZh.value ? {
  docs: '平台文档', contents: '文档目录', login: '登录', eyebrow: 'AIRouter 开发者文档', title: '快速开始',
  intro: '用一个钱包和一个 API Key，接入 AIRouter 的 Codex 模型。本文档覆盖从注册、充值到第一次 API 调用的完整流程。',
  startNow: '从快速开始', updated: '适用于当前 AIRouter 接口', quickStartTitle: '快速开始', quickStartDescription: '按照下面四步完成第一次调用。AIRouter 采用钱包余额按 Token 消耗计费，不提供订阅套餐。',
  tipLabel: '提示：', quickTip: 'API Key 只显示一次，请在创建后立即保存。不同应用建议使用不同 Key，便于单独停用和查看用量。',
  endpointTitle: '接口地址', endpointDescription: '所有 OpenAI 兼容请求都使用同一个 Base URL。客户端会在它后面拼接具体资源路径。', baseUrlLabel: 'OpenAI 兼容 Base URL', responsesLabel: 'Responses API', endpointTableTitle: '常用地址',
  walletTitle: '钱包与计费', walletDescription: '注册奖励、邀请奖励和充值余额统一显示在钱包中，调用产生的费用从可用余额扣除。',
  modelsTitle: '模型与价格', modelsDescription: '价格按每百万 Token 展示，输入和输出分别计费；缓存命中会按缓存价格计费。', model: '模型', positioning: '定位', inputPrice: '输入 / 1M', outputPrice: '输出 / 1M', priceNote: '实际扣费以平台当前渠道定价为准。余额不足时请求会被拒绝，不会产生负余额。',
  examplesTitle: '调用示例', examplesDescription: '下面示例使用 Responses API。将示例中的 API Key 替换为你在控制台创建的 Key。', curlTitle: 'cURL', pythonTitle: 'Python（OpenAI SDK）', copy: '复制', copied: '已复制',
  troubleshootingTitle: '排查与常见问题', troubleshootingDescription: '遇到问题时先确认地址、Key、模型和余额都来自同一个 AIRouter 账户。',
  supportTitle: '联系客服', supportDescription: '遇到接口、充值或账号问题，可通过QQ群或QQ联系我们。', qqGroupLabel: 'QQ群', qqGroupValue: '1109173402', qqLabel: 'QQ', contactNow: '联系QQ',
  footer: 'AIRouter · Codex API 中转服务',
  steps: {
    register: { title: '注册并登录', body: '支持邮箱和 Linux.do 登录。完成注册并登录后，系统会按当前活动规则发放新用户余额。' },
    wallet: { title: '充值钱包', body: '进入控制台的钱包页面，选择充值金额和支付方式。平台不提供订阅，所有服务按实际 Token 用量扣费。' },
    key: { title: '创建 API Key', body: '打开 API 密钥页面，创建一个 Key，并选择可用的 Codex 分组。复制完整 Key 并妥善保存。' },
    client: { title: '配置客户端', body: '把 Base URL 设置为文档中的地址，把 API Key 填入客户端，并使用支持的模型名称发起请求。' }
  },
  walletPoints: [
    { number: '01', title: '余额统一管理', body: '注册奖励、邀请奖励和充值金额都在钱包中查看。' },
    { number: '02', title: '按量扣费', body: '输入 Token、输出 Token 与缓存 Token 按对应价格实时计费。' },
    { number: '03', title: '可查可控', body: '在使用记录中查看每次请求的模型、Token 数量和费用。' }
  ],
  troubleshooting: [
    { title: '返回 401 或 403', body: '确认请求使用的是完整 API Key，Key 没有被删除或禁用，并且请求地址没有多余的 /responses。' },
    { title: '返回余额不足或 402', body: '进入钱包确认可用余额。请求不会因为余额不足而透支，充值后重新发起即可。' },
    { title: '返回 404 或 405', body: 'OpenAI SDK 的 base_url 填写到 /v1，不要把 /responses 再拼到 base_url；Responses 请求由 SDK 调用 /responses。' },
    { title: '请求超时或 503', body: '先检查网络和模型名称，再稍后重试。持续出现时保留请求时间、模型和错误 request id，联系平台管理员。' }
  ]
} : {
  docs: 'Platform Docs', contents: 'Documentation', login: 'Sign in', eyebrow: 'AIRouter Developer Docs', title: 'Quick Start',
  intro: 'Connect to AIRouter Codex models with one wallet and one API key. This guide covers registration, wallet top-up, API key creation, and your first request.',
  startNow: 'Start here', updated: 'For the current AIRouter API', quickStartTitle: 'Quick start', quickStartDescription: 'Complete your first request in four steps. AIRouter uses wallet-based, pay-as-you-go token billing instead of subscriptions.',
  tipLabel: 'Tip: ', quickTip: 'An API key is shown only once. Save it immediately after creation. Use separate keys for separate applications so each can be disabled and monitored independently.',
  endpointTitle: 'Endpoint', endpointDescription: 'All OpenAI-compatible requests use the same Base URL. Your client appends the resource path it needs.', baseUrlLabel: 'OpenAI-compatible Base URL', responsesLabel: 'Responses API', endpointTableTitle: 'Common URLs',
  walletTitle: 'Wallet and billing', walletDescription: 'Registration rewards, invite rewards, and top-ups are shown in the same wallet. Usage is deducted from the available balance.',
  modelsTitle: 'Models and pricing', modelsDescription: 'Prices are shown per million tokens. Input and output are billed separately; cached tokens use their cache rate.', model: 'Model', positioning: 'Positioning', inputPrice: 'Input / 1M', outputPrice: 'Output / 1M', priceNote: 'Actual charges follow the platform pricing currently configured. Requests are rejected when the balance is insufficient; balances do not go negative.',
  examplesTitle: 'Examples', examplesDescription: 'The examples below use the Responses API. Replace the placeholder API key with a key created in your console.', curlTitle: 'cURL', pythonTitle: 'Python (OpenAI SDK)', copy: 'Copy', copied: 'Copied',
  troubleshootingTitle: 'Troubleshooting', troubleshootingDescription: 'When something fails, first confirm that the endpoint, key, model, and wallet all belong to the same AIRouter account.',
  supportTitle: 'Contact support', supportDescription: 'For API, wallet, or account issues, contact us through the QQ group or QQ.', qqGroupLabel: 'QQ group', qqGroupValue: '1109173402', qqLabel: 'QQ', contactNow: 'Contact on QQ',
  footer: 'AIRouter · Codex API relay service',
  steps: {
    register: { title: 'Register and sign in', body: 'Sign up with email or Linux.do. After registration and sign-in, new-account credit is granted according to the current promotion rules.' },
    wallet: { title: 'Top up your wallet', body: 'Open Wallet in the console and choose an amount and payment method. There are no subscriptions; usage is billed by actual token consumption.' },
    key: { title: 'Create an API key', body: 'Open API Keys, create a key, and select an available Codex group. Copy the full key and store it securely.' },
    client: { title: 'Configure your client', body: 'Set the Base URL from this page, add your API key, and send a request using one of the supported model names.' }
  },
  walletPoints: [
    { number: '01', title: 'One balance', body: 'Registration credit, invite rewards, and top-ups are all visible in Wallet.' },
    { number: '02', title: 'Pay as you go', body: 'Input, output, and cached tokens are charged using their respective rates.' },
    { number: '03', title: 'Track every request', body: 'Review model, token counts, and cost in Usage Records.' }
  ],
  troubleshooting: [
    { title: '401 or 403', body: 'Confirm that the full API key is present, has not been deleted or disabled, and that the URL does not contain an extra /responses.' },
    { title: 'Insufficient balance or 402', body: 'Open Wallet and check the available balance. Requests do not overdraft; top up and try again.' },
    { title: '404 or 405', body: 'Set the OpenAI SDK base_url to /v1. Do not append /responses to base_url; the SDK adds that path for Responses requests.' },
    { title: 'Timeout or 503', body: 'Check the network and model name, then retry. If it persists, keep the timestamp, model, and request id for the platform administrator.' }
  ]
})

const sections = computed(() => [
  { id: 'quick-start', title: copy.value.quickStartTitle },
  { id: 'endpoint', title: copy.value.endpointTitle },
  { id: 'wallet', title: copy.value.walletTitle },
  { id: 'models', title: copy.value.modelsTitle },
  { id: 'examples', title: copy.value.examplesTitle },
  { id: 'troubleshooting', title: copy.value.troubleshootingTitle },
  { id: 'support', title: copy.value.supportTitle }
])

const walletPoints = computed(() => copy.value.walletPoints)
const troubleshooting = computed(() => copy.value.troubleshooting)

const endpointRows = computed(() => isZh.value ? [
  { name: 'OpenAI Chat Completions', value: `${baseUrl}/chat/completions` },
  { name: 'OpenAI Responses', value: `${baseUrl}/responses` },
  { name: '模型列表', value: `${baseUrl}/models` }
] : [
  { name: 'OpenAI Chat Completions', value: `${baseUrl}/chat/completions` },
  { name: 'OpenAI Responses', value: `${baseUrl}/responses` },
  { name: 'Model list', value: `${baseUrl}/models` }
])

const models = computed(() => isZh.value ? [
  { name: 'gpt-5.6-sol', positioning: '高阶 / 复杂任务', input: '$6', output: '$36' },
  { name: 'gpt-5.5', positioning: '标准 / 通用任务', input: '$5', output: '$30' },
  { name: 'gpt-5.6-luna', positioning: '经济 / 高频调用', input: '$1', output: '$6' },
  { name: 'gpt-6-astra', positioning: '旗舰 / 高性能', input: '$10', output: '$50' }
] : [
  { name: 'gpt-5.6-sol', positioning: 'Advanced / complex tasks', input: '$6', output: '$36' },
  { name: 'gpt-5.5', positioning: 'Standard / general tasks', input: '$5', output: '$30' },
  { name: 'gpt-5.6-luna', positioning: 'Economy / high-volume calls', input: '$1', output: '$6' },
  { name: 'gpt-6-astra', positioning: 'Flagship / high performance', input: '$10', output: '$50' }
])

const curlExample = [
  `curl ${baseUrl}/responses \\`,
  '  -H "Authorization: Bearer sk-your-api-key" \\',
  '  -H "Content-Type: application/json" \\',
  `  -d '{"model":"gpt-5.6-luna","input":"Hello from AIRouter"}'`
].join('\n')
const pythonExample = computed(() => `from openai import OpenAI

client = OpenAI(
    api_key="sk-your-api-key",
    base_url="${baseUrl}",
)

response = client.responses.create(
    model="gpt-5.6-luna",
    input="Hello from AIRouter",
)
print(response.output_text)`).value

async function copyValue(value: string, key: string) {
  try {
    await navigator.clipboard.writeText(value)
    copiedKey.value = key
    window.setTimeout(() => {
      if (copiedKey.value === key) copiedKey.value = ''
    }, 1600)
  } catch {
    copiedKey.value = ''
  }
}

const SectionHeading = defineComponent({
  props: { number: { type: String, required: true }, title: { type: String, required: true }, description: { type: String, required: true } },
  setup(props) {
    return () => h('div', { class: 'flex items-start gap-3' }, [
      h('span', { class: 'mt-0.5 font-mono text-xs font-semibold text-primary-600 dark:text-primary-400' }, props.number),
      h('div', [h('h2', { class: 'text-xl font-bold text-gray-950 dark:text-white' }, props.title), h('p', { class: 'mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300' }, props.description)])
    ])
  }
})

const StepCard = defineComponent({
  props: { number: { type: String, required: true }, title: { type: String, required: true }, body: { type: String, required: true } },
  setup(props) {
    return () => h('article', { class: 'rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-800 dark:bg-dark-900' }, [
      h('div', { class: 'flex items-start gap-3' }, [
        h('span', { class: 'flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-primary-50 text-sm font-semibold text-primary-700 dark:bg-primary-500/10 dark:text-primary-300' }, props.number),
        h('div', [h('h3', { class: 'text-sm font-semibold text-gray-950 dark:text-white' }, props.title), h('p', { class: 'mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300' }, props.body)])
      ])
    ])
  }
})

const InfoPanel = defineComponent({
  props: { title: { type: String, required: true }, value: { type: String, required: true }, copyLabel: { type: String, required: true }, copied: { type: Boolean, default: false } },
  emits: ['copy'],
  setup(props, { emit }) {
    return () => h('div', { class: 'rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-800 dark:bg-dark-900' }, [
      h('div', { class: 'flex items-center justify-between gap-3' }, [h('span', { class: 'text-sm font-medium text-gray-600 dark:text-dark-300' }, props.title), h('button', { type: 'button', class: 'inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium text-primary-700 hover:bg-primary-50 dark:text-primary-300 dark:hover:bg-primary-500/10', onClick: () => emit('copy') }, [h(Icon, { name: props.copied ? 'check' : 'copy', size: 'xs' }), props.copied ? copy.value.copied : props.copyLabel])]),
      h('code', { class: 'mt-3 block break-all rounded-md bg-gray-50 px-3 py-2 text-xs text-gray-800 dark:bg-dark-800 dark:text-dark-200' }, props.value)
    ])
  }
})

const CodePanel = defineComponent({
  props: { title: { type: String, required: true }, code: { type: String, required: true }, copyLabel: { type: String, required: true }, copied: { type: Boolean, default: false } },
  emits: ['copy'],
  setup(props, { emit }) {
    return () => h('div', { class: 'overflow-hidden rounded-lg border border-gray-200 bg-gray-950 dark:border-dark-700' }, [
      h('div', { class: 'flex items-center justify-between border-b border-white/10 px-4 py-3' }, [h('span', { class: 'text-sm font-medium text-gray-200' }, props.title), h('button', { type: 'button', class: 'inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium text-gray-300 hover:bg-white/10 hover:text-white', onClick: () => emit('copy') }, [h(Icon, { name: props.copied ? 'check' : 'copy', size: 'xs' }), props.copied ? copy.value.copied : props.copyLabel])]),
      h('pre', { class: 'overflow-x-auto p-4 text-xs leading-6 text-gray-200' }, h('code', null, props.code))
    ])
  }
})
</script>
