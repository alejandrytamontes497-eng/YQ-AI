-- Enable the login agreement prompt and seed the WahaAPI legal documents.
-- Only replace login_agreement_documents when the current value is missing or
-- still the old empty default, so customized legal text is preserved.

DO $$
DECLARE
  v_docs jsonb := '[
    {
      "id": "terms",
      "title": "服务条款",
      "content_md": "WahaAPI 服务条款\n生效日期:2026 年 6 月 11 日\n\n使用 WahaAPI 即表示您同意以下条款。WahaAPI 是一项 API 中转服务,将您的请求转发至第三方 AI 模型服务商并返回结果。\n\n1. 服务说明\n我们提供 API 转发,不拥有也不训练上游模型;模型的可用性、性能与价格由上游服务商决定。\n我们可能随时调整、新增或下线部分模型与功能。\n2. 账户与密钥\n请妥善保管 API Key,Key 等同于密码,泄露导致的损失由您自行承担。\n不得转售、共享 Key,或绕过限流、配额等机制。\n我们有权对异常调用、滥用或存在安全风险的账户限流、暂停或封禁。\n3. 内容与隐私\n您的输入会转发给上游服务商处理,并受其政策约束。\n我们不使用您的内容训练模型,仅为计费、安全和排障保留必要的用量记录。\n输出可能不准确或不完整,请自行判断后再使用。\n4. 计费与退款\n采用预付费/按量计费,费率以定价页面为准;余额消耗后不予退还。\n因自身原因(不再需要、计划调整等)申请的退款,我们有权拒绝。\n因上游故障、网络波动导致的请求失败,我们尽力处理但不保证补偿。\n5. 使用限制\n禁止将服务用于:违法内容、网络攻击、垃圾信息、欺诈,或转售、逆向、构建竞品。具体见《使用政策》。违反者将被立即终止服务且不予退款。\n\n6. 免责声明\n服务按\"现状\"提供,不保证可用性、准确性或连续性。在法律允许范围内,我们对间接损失不承担责任,累计赔偿以您最近 12 个月支付的费用为上限。\n\n7. 服务区域\n本服务仅向《支持区域政策》列明的国家和地区开放,受制裁地区除外。\n\n8. 条款变更\n我们可随时更新本条款,更新后于网站公布即生效,继续使用即视为接受。"
    },
    {
      "id": "usage-policy",
      "title": "使用政策",
      "content_md": "可接受使用政策(Acceptable Use Policy)\n使用本服务时,禁止将其用于以下任何用途:\n\n1.禁止的内容\n生成或传播违法、暴力、恐怖主义、儿童性化(CSAM)等内容\n生成仇恨、骚扰、人身威胁或歧视性内容\n制造虚假信息、欺诈、网络钓鱼或冒充他人\n2.禁止的行为\n任何形式的网络攻击、恶意软件生成、漏洞利用或未授权入侵\n大规模垃圾信息、自动化骚扰或滥发(spam)\n绕过、破解或攻击本服务及上游服务商的限流与安全机制\n逆向工程、抓取本服务接口以重建竞争性服务\n侵犯他人知识产权、隐私权或商业秘密\n3.数据与合规\n用户须对其上传/提交的数据及输出结果承担全部法律责任\n不得提交受出口管制、制裁或保密义务约束而未获授权的数据\n用户须遵守上游模型服务商(如 OpenAI、Anthropic 等)的使用政策\n4.违规处理\n违反上述政策的账户将被立即暂停或封禁,且不予退款;涉嫌违法的,本服务将依法配合相关部门调查。"
    },
    {
      "id": "supported-regions",
      "title": "支持的国家和地区",
      "content_md": "支持的国家和地区(Supported Regions)\n本服务面向以下国家和地区的用户开放(不包括中国大陆)。\n\n注:出于合规与上游服务商可用性考虑,中国大陆地区不在服务范围内。\n\n亚太地区\n\n中国香港、中国澳门、中国台湾\n日本、韩国、新加坡、马来西亚、泰国、越南、菲律宾、印度尼西亚\n印度、澳大利亚、新西兰\n北美洲\n\n美国、加拿大、墨西哥\n欧洲\n\n英国、爱尔兰、德国、法国、荷兰、比利时、卢森堡\n西班牙、葡萄牙、意大利、瑞士、奥地利\n瑞典、挪威、丹麦、芬兰、冰岛\n波兰、捷克、匈牙利、希腊、罗马尼亚\n中东与非洲\n\n阿联酋、以色列、沙特阿拉伯、卡塔尔\n南非\n南美洲\n\n巴西、阿根廷、智利\n上述列表仅供参考,本服务保留根据合规要求随时调整支持地区的权利。受国际制裁的国家/地区(如朝鲜、伊朗、叙利亚、古巴、俄罗斯部分地区等)不在服务范围内。"
    },
    {
      "id": "service-specific-terms",
      "title": "服务特定条款",
      "content_md": "WahaAPI 服务特定条款\n生效日期:2026 年 5 月 1 日\n\n这些服务特定条款(\"服务特定条款\")包含适用于特定服务的补充条款,并构成客户与 WahaAPI 之间就引用这些服务特定条款的服务(例如《商业服务条款》)的协议的一部分(\"协议\")。本文件中未定义的大写术语具有协议中规定的含义。如条款与本服务特定条款存在冲突,以本服务特定条款为准。\n\nA. 服务性质(API 中转服务)\nWahaAPI 是一项 API 中转与代理转发服务,通过统一接口将客户请求转发至第三方上游 AI 模型服务商(\"上游服务商\",例如 OpenAI、Anthropic 等)并返回响应。客户承认并同意:\n\nWahaAPI 不拥有、不开发、不训练上游模型,仅提供请求的转发、聚合与管理能力。\n模型的可用性、性能、定价、输出质量及行为完全由上游服务商决定,WahaAPI 不作任何保证。\n客户对模型的使用同时受上游服务商各自的使用条款与政策约束,客户须自行了解并遵守。\n上游服务商对其服务的任何变更、限流、调价或下线,可能直接影响客户对本服务的使用,WahaAPI 对此不承担责任。\nB. 客户内容与数据流转\n客户提交的输入将被转发至所选上游服务商以生成输出。客户理解其内容将经由上游服务商处理,并受其数据政策约束。\nWahaAPI 不使用客户内容训练任何模型。\n除为提供服务、计费、安全风控及法律合规所必需外,WahaAPI 不会留存客户的请求内容;具体保留规则见下文第 F 节及 DPA。\nC. Beta 服务\nWahaAPI 可能提供处于预发布、测试或预览阶段的服务(\"Beta 服务\"),按\"原样\"临时提供,不适合生产使用。WahaAPI 不对客户使用或依赖 Beta 服务承担责任,亦无赔偿义务。WahaAPI 对 Beta 服务的责任以客户在前 12 个月内为服务支付的费用与 $1,000 中的较小者为上限。\n\nD. API 使用限制\n客户必须遵守以下限制,WahaAPI 有权对违反者限流、暂停或封禁:\n\n速率限制: 客户须遵守 WahaAPI 及上游服务商规定的 API 调用速率限制(RPM/TPM 等)。\n配额: 客户使用不得超过其套餐或余额所对应的分配配额。\n并发: 客户须遵守其计划允许的最大并发请求数。\n缓存: 客户可缓存响应,但须遵守 WahaAPI 的缓存政策,且不得借此规避计费。\n滥用防护: 客户不得通过脚本、批量注册、共享密钥等方式绕过限流、配额或风控机制。\n监控: WahaAPI 保留监控 API 使用情况以保障服务安全与稳定的权利。\nE. 密钥与账户安全\n客户对其 API Key 及账户下的所有活动负全部责任。\nAPI Key 等同于凭证,客户须妥善保管,不得公开泄露、上传至公共代码仓库或转售给未授权第三方。\n因 Key 泄露、被盗用、被滥用而产生的一切费用与损失由客户承担。\n客户发现账户异常、Key 泄露或遭受攻击时,须立即通知 WahaAPI;WahaAPI 可立即冻结相关 Key 以止损。\nF. 服务级别(SLA)\nWahaAPI 致力于提供高可用服务,但作为依赖上游的中转服务,可用性同时取决于上游服务商:\n\n可用性目标: 在 WahaAPI 自身可控范围内,目标为 99.9% 的月度正常运行时间。\n不计入停机: 因上游服务商故障、限流、封禁,以及计划内维护、不可抗力、客户自身原因导致的不可用,不计入 SLA 停机时间。\n维护: 计划内维护将尽合理努力提前通知。\n支持: 按客户所选计划提供相应级别的技术支持。\nG. 退款与计费(补充)\n本服务采用预付费/按量计费,费率以模型定价页面为准。\n因上游服务商故障、网络波动等非 WahaAPI 直接原因导致的请求失败,WahaAPI 将尽合理努力处理但不保证补偿;具体以实际情况为准。\n客户应自行监控用量,避免因异常调用导致余额快速消耗;由此产生的费用由客户承担。\nH. 合规与禁止用途\n客户使用本服务须遵守《使用政策》《支持区域政策》及上游服务商政策,不得用于任何违法、滥用或被禁止的用途。违反者 WahaAPI 有权立即暂停或终止服务且不予退款。\n\nI. 更新和修改\nWahaAPI 可不时更新本服务特定条款。重大更改将提前通知客户,并于通知后 30 天生效;为响应法律法规或上游服务商政策变化而作的更新可在发布或通知后立即生效。"
    }
  ]'::jsonb;
  v_old_empty jsonb := '[
    {"id":"terms","title":"服务条款","content_md":""},
    {"id":"usage-policy","title":"使用政策","content_md":""},
    {"id":"supported-regions","title":"支持的国家和地区","content_md":""},
    {"id":"service-specific-terms","title":"服务特定条款","content_md":""}
  ]'::jsonb;
  v_current text;
  v_current_json jsonb;
  v_should_seed boolean := false;
BEGIN
  SELECT value INTO v_current FROM settings WHERE key = 'login_agreement_documents';

  IF v_current IS NULL OR btrim(v_current) = '' THEN
    v_should_seed := true;
  ELSE
    BEGIN
      v_current_json := v_current::jsonb;
      v_should_seed := v_current_json = v_old_empty;
    EXCEPTION WHEN others THEN
      v_should_seed := true;
    END;
  END IF;

  IF v_should_seed THEN
    INSERT INTO settings (key, value, updated_at)
    VALUES ('login_agreement_documents', v_docs::text, NOW())
    ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW();

    INSERT INTO settings (key, value, updated_at)
    VALUES ('login_agreement_enabled', 'true', NOW())
    ON CONFLICT (key) DO UPDATE SET value = 'true', updated_at = NOW();

    INSERT INTO settings (key, value, updated_at)
    VALUES ('login_agreement_mode', 'modal', NOW())
    ON CONFLICT (key) DO UPDATE SET value = 'modal', updated_at = NOW();

    INSERT INTO settings (key, value, updated_at)
    VALUES ('login_agreement_updated_at', '2026-06-10', NOW())
    ON CONFLICT (key) DO UPDATE SET value = '2026-06-10', updated_at = NOW();
  END IF;
END
$$;
