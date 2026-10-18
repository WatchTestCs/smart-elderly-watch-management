<template>
  <div class="device">
    <header class="collection-heading"><div><h1>设备管理</h1><p>查看连接状态、跟进异常，并为家人关联照护设备。</p></div><el-button type="primary" aria-label="登记新设备" icon="el-icon-plus" @click="createDialogVisible = true">登记新设备</el-button></header>
    <section class="collection-metrics" aria-label="设备摘要"><article><span>登记设备</span><strong>{{ deviceLoadError && !deviceList.length ? '—' : deviceTotal }}</strong><small>当前筛选中的设备总数</small></article><article><span>在线设备</span><strong>{{ deviceLoadError && !deviceList.length ? '—' : onlineCount }}</strong><small>以最近一次设备连接状态为准</small></article><article><span>待处理异常</span><strong :class="alertsReady && alertTotals.total ? 'attention-number' : ''">{{ alertsReady ? alertTotals.total : '—' }}</strong><small>{{ alertsReady ? '人工确认与跟进后更新状态' : '异常统计暂未就绪' }}</small></article></section>
    <div v-if="deviceLoadError || alertLoadError" class="collection-feedback" role="alert"><span>{{ deviceLoadError || alertLoadError }}</span><el-button plain :loading="deviceLoading" @click="loadDevices">重新加载设备</el-button></div>
    <section class="device-alert-summary"><div><h2>异常跟进</h2><p v-if="alertsReady">人员 {{ alertTotals.personTotal }} · 心率 {{ alertTotals.rateTotal }} · 血压 {{ alertTotals.preTotal }}</p><p v-else>异常统计暂时无法确认，请重新加载。</p></div><el-button plain :disabled="!alertsReady || alertTotals.total === 0" @click="openWorkbench">查看全部待处理</el-button></section>
    <section class="collection-panel">
      <div class="collection-toolbar"><div class="collection-filters"><div><label for="device-query">查找设备</label><el-input id="device-query" v-model="deviceQuery" clearable placeholder="设备号、别名或长者姓名"></el-input></div><div><label for="device-state">连接状态</label><el-select id="device-state" v-model="deviceState"><el-option label="全部设备" value="all"></el-option><el-option label="在线" value="online"></el-option><el-option label="离线" value="offline"></el-option></el-select></div></div><div class="collection-tools"><el-button plain :loading="deviceLoading" @click="loadDevices">刷新</el-button><el-switch v-model="autoRefresh" active-text="自动刷新" aria-label="自动刷新设备列表"></el-switch></div></div>
      <p class="collection-caption" role="status">{{ autoRefresh ? '每5秒自动刷新' : '自动刷新已关闭' }} · 当前显示 {{ visibleDevices.length }} 台设备</p>
    <el-table class="device-desktop-table" v-loading="deviceLoading" :data="visibleDevices" style="width:100%">
      <template #empty><div  class="collection-empty"><strong>{{ deviceLoadError ? '设备列表暂时无法加载' : deviceList.length ? '没有匹配的设备' : '还没有登记设备' }}</strong><span>调整关键词或登记一台新的照护手表。</span></div></template>
      <el-table-column label="照护设备" min-width="180"><template #default="scope" ><strong class="device-table-name">{{ scope.row.alias || '未命名设备' }}</strong><small class="device-table-secondary">{{ scope.row.deviceNo }}</small></template></el-table-column>
      <el-table-column label="关联长者" min-width="105"><template #default="scope" >{{ scope.row.oldPersonName || '尚未绑定' }}</template></el-table-column>
      <el-table-column label="连接状态" width="90"><template #default="scope" ><el-tag v-if="scope.row.online === 1 || scope.row.online === '1'" type="success">在线</el-tag><el-tag v-else-if="scope.row.online === 0 || scope.row.online === '0'" type="info">离线</el-tag><span v-else>状态未知</span></template></el-table-column>
      <el-table-column label="待处理异常" width="210"><template #default="scope" ><div class="device-table-alerts"><el-button link :class="alertOf(scope.row.deviceId).personCount ? 'has-alert' : ''" :disabled="!alertsReady" @click="showAlert(scope.row, 'person')">人员 {{ alertsReady ? alertOf(scope.row.deviceId).personCount : '—' }}</el-button><el-button link :class="alertOf(scope.row.deviceId).rateCount ? 'has-alert' : ''" :disabled="!alertsReady" @click="showAlert(scope.row, 'rate')">心率 {{ alertsReady ? alertOf(scope.row.deviceId).rateCount : '—' }}</el-button><el-button link :class="alertOf(scope.row.deviceId).preCount ? 'has-alert' : ''" :disabled="!alertsReady" @click="showAlert(scope.row, 'pre')">血压 {{ alertsReady ? alertOf(scope.row.deviceId).preCount : '—' }}</el-button></div></template></el-table-column>
      <el-table-column label="电量" width="75"><template #default="scope" >{{ scope.row.battery == null ? '暂无' : scope.row.battery + '%' }}</template></el-table-column>
      <el-table-column label="最近上报" width="155"><template #default="scope" ><span class="device-table-time">{{ scope.row.lastTime || '暂无记录' }}</span></template></el-table-column>
      <el-table-column label="操作" width="190" fixed="right"><template #default="scope" ><div class="device-table-actions"><el-button type="primary" plain @click="openBindCodeDialog(scope.row)">绑定码</el-button><el-dropdown trigger="click" @command="handleDeviceCommand($event, scope.row)"><el-button plain aria-label="更多设备操作">更多 <i class="el-icon-arrow-down" aria-hidden="true"></i></el-button><template #dropdown><el-dropdown-menu ><el-dropdown-item command="details">设备详情</el-dropdown-item><el-dropdown-item command="reset">重置密钥</el-dropdown-item><el-dropdown-item command="unbind" :disabled="!scope.row.oldId">解除绑定</el-dropdown-item><el-dropdown-item command="delete" divided>删除设备</el-dropdown-item></el-dropdown-menu></template></el-dropdown></div></template></el-table-column>
    </el-table>
    <el-pagination :current-page="currentPage" :page-size="pageSize" :total="deviceTotal" layout="total, prev, pager, next" @current-change="changeDevicePage" />
    <div v-if="visibleDevices.length" class="device-mobile-grid"><article v-for="item in visibleDevices" :key="item.deviceId" class="device-record"><header><div><h3>{{ item.alias || '未命名设备' }}</h3><p>{{ item.deviceNo }}</p></div><span class="connection-state" :class="item.online == 1 ? 'is-online' : ''">{{ item.online == 1 ? '在线' : item.online == 0 ? '离线' : '状态未知' }}</span></header><dl><dt>绑定长者</dt><dd>{{ item.oldPersonName || '尚未绑定' }}</dd><dt>设备编号</dt><dd>{{ item.deviceId }}</dd><dt>剩余电量</dt><dd>{{ item.battery == null ? '暂无数据' : item.battery + '%' }}</dd><dt>最后上报</dt><dd>{{ item.lastTime || '暂无记录' }}</dd><dt>位置坐标</dt><dd>{{ item.lastLng == null || item.lastLat == null ? '暂无数据' : item.lastLng + '，' + item.lastLat }}</dd></dl><div class="device-alert-actions"><el-button plain :disabled="!alertsReady" @click="showAlert(item, 'person')">人员 · {{ alertsReady ? alertOf(item.deviceId).personCount : '—' }}</el-button><el-button plain :disabled="!alertsReady" @click="showAlert(item, 'rate')">心率 · {{ alertsReady ? alertOf(item.deviceId).rateCount : '—' }}</el-button><el-button plain :disabled="!alertsReady" @click="showAlert(item, 'pre')">血压 · {{ alertsReady ? alertOf(item.deviceId).preCount : '—' }}</el-button></div><div class="device-record-actions"><el-button type="primary" plain @click="openBindCodeDialog(item)">设备绑定码</el-button><el-button plain @click="resetSecret(item)">重置密钥</el-button><el-button plain :disabled="!item.oldId" @click="unbind(item)">解除绑定</el-button><el-button plain type="danger" @click="deleteDevice(item)">删除设备</el-button></div></article></div>
    <div v-else class="device-mobile-empty collection-empty"><strong>{{ deviceLoading ? '正在加载设备…' : deviceLoadError ? '设备列表暂时无法加载' : deviceList.length ? '没有匹配的设备' : '还没有登记设备' }}</strong><p>{{ deviceQuery || deviceState !== 'all' ? '调整关键词或连接状态，即可查看其他设备。' : '登记第一台手表，让照护信息开始连接。' }}</p></div>
    </section>
    <el-dialog title="设备详情" v-model="deviceDetailVisible"><dl v-if="detailDevice" class="device-detail-grid"><dt>设备别名</dt><dd>{{ detailDevice.alias || '未命名设备' }}</dd><dt>设备号</dt><dd>{{ detailDevice.deviceNo }}</dd><dt>设备编号</dt><dd>{{ detailDevice.deviceId }}</dd><dt>设备型号</dt><dd>{{ detailDevice.model || '暂无信息' }}</dd><dt>绑定长者</dt><dd>{{ detailDevice.oldPersonName || '尚未绑定' }}</dd><dt>最近上报</dt><dd>{{ detailDevice.lastTime || '暂无记录' }}</dd><dt>位置坐标</dt><dd>{{ detailDevice.lastLng == null || detailDevice.lastLat == null ? '暂无数据' : detailDevice.lastLng + '，' + detailDevice.lastLat }}</dd></dl><template #footer><span ><el-button type="primary" @click="deviceDetailVisible = false">关闭详情</el-button></span></template></el-dialog>
    <aside class="device-onboarding"><img src="/static/img/care-courtyard.webp" alt="" width="960" height="411"><div><p class="collection-eyebrow">从一次连接开始</p><h2>让设备，成为关爱的延伸</h2><ol><li>登记手表，获取设备号与密钥。</li><li>在手表端配置连接信息。</li><li>生成绑定码，关联需要照护的长者。</li></ol><p class="collection-caption">设备注册后，请以实际上传记录确认连接是否正常。</p></div></aside>
    
    <el-dialog :title="alertDialog.title" v-model="alertDialog.visible" width="620px">
      <div v-if="alertDialog.loading" style="text-align:center;padding:20px;">正在读取 ...</div>
      <div v-else-if="alertDialog.hasAlert === 0" class="no-alert">
        <i class="el-icon-success" style="color:#52BAB5;font-size:32px;"></i>
        <div style="margin-top:8px;">{{ alertDialog.message }}</div>
      </div>
      <div v-else>
        <el-alert :title="'问题：' + alertDialog.reason" type="error" :closable="false" show-icon></el-alert>

        <table class="alert-table">
          <tbody>
            <tr><th>设备</th><td>{{ alertDialog.deviceNo }}（{{ alertDialog.alias }}）</td></tr>
            <tr><th>绑定老人</th><td>{{ alertDialog.oldPersonName }}</td></tr>
            <tr v-if="alertDialog.value !== ''"><th>本次测量</th><td>{{ alertDialog.value }}</td></tr>
            <tr v-if="alertDialog.threshold"><th>正常范围</th><td>{{ alertDialog.threshold }}</td></tr>
            <tr><th>发生时间</th><td>{{ alertDialog.occurTime }}</td></tr>
            <tr><th>判断依据</th><td>{{ alertDialog.detail }}</td></tr>
            <tr><th>怎么处理</th><td>{{ alertDialog.advice }}</td></tr>
          </tbody>
        </table>

        <div class="alert-actions">
          <el-button type="primary" :loading="alertDialog.resolving" @click="resolveAlert">标记已处理（消除红点）</el-button>
          <el-button plain @click="gotoPage(alertDialog.kind)">去相关页面看看</el-button>
        </div>
        <div class="tip">
          「标记已处理」表示这个问题已经人工确认/跟进过了：设备类问题会记录为已处理，
          心率血压会同时把这条记录从"异常"改为"已处理"，红点随之消失，历史记录仍然保留。
        </div>
      </div>
      <template #footer><span  class="dialog-footer">
        <el-button @click="alertDialog.visible = false">关 闭</el-button>
      </span></template>
    </el-dialog>

    <el-dialog title="待处理异常" v-model="workbench.visible" width="760px">
      <div class="tip">共 {{ workbench.list.length }} 条未处理异常。点「查看原因」看具体问题和处理办法，处理完点「标记已处理」消除红点。</div>
      <el-table :data="workbench.list" size="small" border style="width:100%;margin-top:8px;">
        <el-table-column prop="deviceNo" label="设备号" width="120" align="center"></el-table-column>
        <el-table-column prop="alias" label="别名" width="110" align="center"></el-table-column>
        <el-table-column label="问题" width="120" align="center">
          <template #default="scope" >
            <el-tag type="danger" size="small">{{ scope.row.kindText }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="reason" label="原因" align="left"></el-table-column>
        <el-table-column label="操作" width="170" align="center">
          <template #default="scope" >
            <el-button size="small" plain type="primary" @click="showAlert(scope.row, scope.row.kind)">查看原因</el-button>
            <el-button size="small" plain @click="resolveAlertRow(scope.row)">已处理</el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #footer><span  class="dialog-footer">
        <el-button @click="workbench.visible = false">关 闭</el-button>
      </span></template>
    </el-dialog>

    <el-dialog title="登记新设备" v-model="createDialogVisible" width="560px" :close-on-click-modal="!creating" :close-on-press-escape="!creating" :show-close="!creating" @closed="closeCreateDialog">
      <el-form label-width="100px" :model="createForm" ref="createFormRef">
        <el-form-item label="别名">
          <el-input id="device-alias" v-model="createForm.alias" placeholder="如：爷爷的手表" maxlength="16"></el-input>
        </el-form-item>
        <el-form-item label="型号" for="device-model">
          <el-input id="device-model" v-model="createForm.model" placeholder="如：HarmonyOS-Wear"></el-input>
        </el-form-item>
        <el-form-item label="直接绑定老人">
          <el-select v-model="createForm.oldId" placeholder="可不选，之后用绑定码绑定" clearable style="width: 100%;">
            <el-option v-for="item in oldPeople" :key="item.oldId"
                       :label="item.name + '（编号 ' + item.oldId + '）'" :value="item.oldId"></el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <div v-if="createdDevice" class="created">
        <div class="created-title">设备已登记，请把下面的信息填到手表端：</div>
        <div class="created-line">设备号：<b>{{ createdDevice.deviceNo }}</b>
          <el-button size="small" plain type="primary" @click="copy(createdDevice.deviceNo)">复制</el-button>
        </div>
        <div class="created-line">密　钥：<b class="secret">{{ createdDevice.secret }}</b>
          <el-button size="small" plain type="primary" @click="copy(createdDevice.secret)">复制</el-button>
        </div>
      </div>
      <template #footer><span  class="dialog-footer">
        <el-button :disabled="creating" @click="closeCreateDialog">关 闭</el-button>
        <el-button v-if="!createdDevice" type="primary" :loading="creating" @click="createDevice">登 记</el-button>
      </span></template>
    </el-dialog>

    <el-dialog title="设备绑定码" v-model="bindCodeDialogVisible" width="560px">
      <div v-if="currentDevice" class="tip">
        设备：{{ currentDevice.deviceNo }}（{{ currentDevice.alias }}）；
        当前绑定老人：{{ currentDevice.oldPersonName }}
      </div>
      <el-form :inline="true" label-width="80px">
        <el-form-item label="绑定老人">
          <el-select v-model="bindOldId" placeholder="请选择老人" style="width: 160px;">
            <el-option v-for="item in oldPeople" :key="item.oldId"
                       :label="item.name + '（编号 ' + item.oldId + '）'" :value="item.oldId"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="有效天数">
          <el-input v-model="bindExpireDays" style="width: 90px;"></el-input>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="createBindCode">生成绑定码</el-button>
        </el-form-item>
      </el-form>

      <div v-if="newBindCode" class="new-code">
        <div>生成成功，请在手表上输入：</div>
        <div class="code-text">{{ newBindCode.code }}
          <el-button size="small" plain type="primary" @click="copy(newBindCode.code)">复制</el-button>
        </div>
        <div class="tip">有效期至 {{ newBindCode.expireTime }}，一个绑定码只能绑定一台手表</div>
      </div>

      <el-table :data="bindCodeList" size="small" border style="width: 100%;margin-top: 10px;">
        <el-table-column prop="code" label="绑定码" width="110" align="center"></el-table-column>
        <el-table-column prop="expireTime" label="过期时间" width="160" align="center"></el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template #default="scope" >
            <el-tag v-if="scope.row.used === 1" type="info" size="small">已使用</el-tag>
            <el-tag v-else type="success" size="small">未使用</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" align="center">
          <template #default="scope" >
            <el-button size="small" plain type="danger" :disabled="scope.row.used === 1"
                       @click="deleteBindCode(scope.row)">作废</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script src="./device.logic.js"></script>

<style scoped src="./device.styles.css"></style>
