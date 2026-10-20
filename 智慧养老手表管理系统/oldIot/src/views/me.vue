<template>
  <div class="profile-page">
    <el-alert v-if="profileLoadError" :title="profileLoadError" type="error" :closable="false" show-icon></el-alert>
    <el-button v-if="profileLoadError" link @click="searchFamily">重新加载资料</el-button>
    <SecurityPanel />
    <section v-loading="isProfileLoading" class="profile-card" aria-label="个人资料">
      <div class="profile-summary">
        <div class="profile-avatar"><img v-if="imageUrl" :src="imageUrl" alt="当前账号头像" @error="imageUrl = ''"><span v-else>{{ (famiyList.familyName || personName || '家').slice(0, 1) }}</span></div>
        <div class="profile-identity"><p>家属账号</p><h2>{{ famiyList.familyName || personName || '个人资料' }}</h2><span>在这里管理您的联系资料与账号。</span></div>
      </div>
      <div class="profile-actions"><el-button type="primary" @click="handleeditMessage">修改资料</el-button><el-button plain @click="handleeditPassword">修改密码</el-button><el-upload class="avatar-uploader" action="restful/image/saveImage" :headers="$uploadHeaders" :show-file-list="false" :with-credentials="true" :on-success="handleAvatarSuccess" :on-error="handleAvatarError" :before-upload="beforeAvatarUpload"><el-button plain>更换头像</el-button></el-upload></div>
      <dl class="profile-details"><div><dt>姓名</dt><dd>{{ famiyList.familyName || '暂未填写' }}</dd></div><div><dt>性别</dt><dd>{{ famiyList.familySex || '暂未填写' }}</dd></div><div><dt>手机号</dt><dd>{{ familyPhone || '暂未提供' }}</dd></div><div><dt>联系地址</dt><dd>{{ famiyList.familyAddress || '暂未填写' }}</dd></div></dl>
    </section>
<el-dialog
				  title="编辑资料"
				  v-model="messageEditDialogVisible"
          :close-on-click-modal="!isProfileSaving" :close-on-press-escape="!isProfileSaving" :show-close="!isProfileSaving"
				  width="92%" custom-class="profile-dialog">
				  <el-form label-width="80px" ref="profileEditForm" :rules="profileRules" :model="messageEditForm">
				    <el-form-item label="家属ID" prop="familyId" style="display: none; ">
				    <el-input v-model="messageEditForm.familyId"></el-input>
				    </el-form-item>
				    <el-form-item label="姓名" for="profile-name" prop="familyName" >
				    <el-input id="profile-name" maxlength="10" autocomplete="name" aria-label="姓名" v-model="messageEditForm.familyName"></el-input>
				    </el-form-item>
				    <el-form-item label="性别" for="profile-sex" prop="familySex">
				    <el-select popper-class="profile-sex-options" id="profile-sex" aria-label="性别" v-model="messageEditForm.familySex"><el-option label="男" value="男"></el-option><el-option label="女" value="女"></el-option></el-select>
				    </el-form-item>
						<el-form-item label="地址" for="profile-address" prop="familyAddress">
						<el-input id="profile-address" maxlength="20" autocomplete="street-address" aria-label="联系地址" v-model="messageEditForm.familyAddress"></el-input>
						</el-form-item>
						<el-form-item label="电话" for="profile-phone" prop="familyPhone">
						<el-input id="profile-phone" aria-label="手机号" v-model="messageEditForm.familyPhone" disabled="disabled"></el-input>
						</el-form-item>
				  </el-form>
				  <template #footer><span  class="dialog-footer">
				  <el-button :disabled="isProfileSaving" @click="messageEditDialogVisible = false">取 消</el-button>
				  <el-button type="primary" :loading="isProfileSaving" @click="editFamily">确 定</el-button>
				</span></template>
				</el-dialog>

				<el-dialog
				  title="修改密码"
				  v-model="passwordEditDialogVisible"
          :close-on-click-modal="!isPasswordSaving" :close-on-press-escape="!isPasswordSaving" :show-close="!isPasswordSaving" @closed="resetForm('passwordEditForm')"
				  width="92%" custom-class="profile-dialog">
				  <el-form label-width="80px" class="demo-ruleForm" status-icon :rules="rules2" ref="passwordEditForm" :model="passwordEditForm">
				    <el-form-item label="" prop="familyPhone" style="display: none; ">
				    <el-input v-model="passwordEditForm.familyPhone"></el-input>
				    </el-form-item>
				    <el-form-item label="原密码" for="profile-old-password" prop="oldPassword" >
				    <el-input id="profile-old-password" type="password" aria-label="原密码" autocomplete="current-password" v-model="passwordEditForm.oldPassword"></el-input>
				    </el-form-item>
				    <el-form-item label="密码" for="profile-new-password" prop="familyPassword">
							<el-input id="profile-new-password" type="password" aria-label="新密码" autocomplete="new-password" v-model="passwordEditForm.familyPassword"></el-input>
						</el-form-item>
						<el-form-item label="确认密码" for="profile-confirm-password" prop="familyPassword2">
							<el-input id="profile-confirm-password" type="password" aria-label="确认密码" autocomplete="new-password" v-model="passwordEditForm.familyPassword2"></el-input>
						</el-form-item>
				  </el-form>
				  <template #footer><span  class="dialog-footer">
				  <el-button :disabled="isPasswordSaving" @click="resetForm('passwordEditForm')">取 消</el-button>
				  <el-button type="primary" :loading="isPasswordSaving" @click="editPassword('passwordEditForm')">确 定</el-button>
				</span></template>
				</el-dialog>

			  </div>
</template>
<style scoped>
.profile-card { padding: 32px; border: 1px solid #dfe5d8; border-radius: 16px; background: #fff; }
.profile-summary { display: flex; align-items: center; gap: 22px; }
.profile-avatar { display: grid; place-items: center; width: 88px; height: 88px; flex: 0 0 88px; overflow: hidden; border-radius: 50%; background: #e6eddd; color: #235442; font-size: 34px; font-weight: 600; }
.profile-avatar img { width: 100%; height: 100%; object-fit: cover; }
.profile-identity { min-width: 0; overflow-wrap: anywhere; }
.profile-identity p { margin: 0 0 6px; color: #59665f; font-size: 16px; }
.profile-identity h2 { margin: 0 0 6px; color: #183e33; font-size: 28px; }
.profile-identity span { color: #53645c; font-size: 17px; line-height: 1.7; }
.profile-actions { display: flex; flex-wrap: wrap; gap: 12px; margin: 28px 0; }
.profile-actions :deep(.el-button) { margin: 0; }
.profile-details { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 32px; margin: 0; }
.profile-details div { padding: 20px 0; border-top: 1px solid #e8ece4; }
.profile-details dt { color: #59665f; font-size: 16px; }
.profile-details dd { margin: 6px 0 0; color: #263632; font-size: 19px; line-height: 1.7; overflow-wrap: anywhere; }
@media (max-width: 600px) { .profile-card { padding: 24px; } .profile-summary { align-items: flex-start; gap: 16px; } .profile-avatar { width: 64px; height: 64px; flex-basis: 64px; font-size: 28px; } .profile-identity h2 { font-size: 24px; } .profile-actions { gap: 10px; } .profile-actions :deep(.el-button) { width: 100%; } .profile-actions > * { flex: 1 0 100%; } .profile-actions :deep(.el-upload) { display: block; } .profile-details { grid-template-columns: 1fr; } }
html.large-text .care-family .profile-identity span, html.large-text .care-family .profile-details dd { font-size: 21px; }
</style>
<script>

	import axios from 'axios'
import SecurityPanel from '../components/SecurityPanel.vue'

export default {
  components: { SecurityPanel },
	data() {

      var validatePass = (rule, value, callback) => {
        if (value === '') {
          callback(new Error('请输入密码'));
        } else {
          if (this.passwordEditForm.familyPassword2 !== '') {
            this.$refs.passwordEditForm.validateField('familyPassword2');
          }
          callback();
        }
      };
      var validatePass2 = (rule, value, callback) => {
        if (value === '') {
          callback(new Error('请再次输入密码'));
        } else if (value !== this.passwordEditForm.familyPassword) {
          callback(new Error('两次输入密码不一致!'));
        } else {
          callback();
        }
      };
      return {
        imageUrl: '',
        isProfileLoading: false,
        isProfileSaving: false,
        profileLoadError: '',
        isPasswordSaving: false,
				messageEditDialogVisible:false,
				passwordEditDialogVisible:false,
				messageEditForm: {
				  familyId: '',
				  familyPhone: '',
          familyName: '', familySex: '', familyAddress: ''
				},
				passwordEditForm: {
				  familyPhone: '',
				  familyPassword: '',
					familyPassword2:'',
					oldPassword: ''
				},
				personName:'',
				familyPhone:'',
				familyTouPic:'',
				famiyList: [],
				profileRules: {
          familyName: [{ required: true, whitespace: true, message: '请输入姓名', trigger: 'blur' }, { max: 10, message: '姓名最多10个字符', trigger: 'blur' }],
          familySex: [{ required: true, message: '请选择性别', trigger: 'change' }],
          familyAddress: [{ max: 20, message: '地址最多20个字符', trigger: 'blur' }]
        },
        rules2: {
          oldPassword: [{ required: true, message: '请输入原密码', trigger: 'blur' }],
				    familyPassword: [
				      { validator: validatePass, trigger: 'blur' },
							{ pattern: /^(?=.*[A-Za-z])(?=.*[0-9]).{12,20}$/, message: '请设置 12–20 位含字母和数字的密码', trigger: 'blur' }
				    ],
				    familyPassword2: [
				      { validator: validatePass2, trigger: 'blur' },
							{ pattern: /^(?=.*[A-Za-z])(?=.*[0-9]).{12,20}$/, message: '请设置 12–20 位含字母和数字的密码', trigger: 'blur' }
				    ]
				  }
				}

    },
		created () {
		  this.personName=localStorage.getItem('user_name')
			this.familyPhone=localStorage.getItem('familyPhone')
			this.searchFamily()

		},
    methods: {
			searchFamily(){
			  var _this = this
          _this.isProfileLoading = true
          _this.profileLoadError = ''
			  axios.post('restful/family/selectTwo',
			    {
			      'familyPhone': _this.familyPhone
			    },
			    {
			      headers: {
			        'content-type': 'application/json'
			      },
			      withCredentials: true
			    }).then(function (response) {

			    _this.famiyList = response.data && response.data.familyDto2 ? response.data.familyDto2 : {}
					localStorage.setItem('user_name', _this.famiyList.familyName)
					localStorage.setItem('familyTouPic', _this.famiyList.familyTouPic)
					_this.familyTouPic=localStorage.getItem('familyTouPic')
					_this.imageUrl = _this.familyTouPic && _this.familyTouPic !== 'undefined' && _this.familyTouPic !== 'null' ? 'restful/image/download?name=' + encodeURIComponent(_this.familyTouPic) : ''

			  })
			    .catch(function (error) {

              _this.profileLoadError = '个人资料暂时无法加载，请重试。'
			    }).then(function () { _this.isProfileLoading = false })
			},
      editFamily () {
        if (this.isProfileSaving) return
        this.$refs.profileEditForm.validate((valid) => {
          if (!valid) return
          this.isProfileSaving = true
          axios.post('restful/family/updateTwo', this.messageEditForm, { withCredentials: true }).then((response) => {
            if (response.data.returnCode === '1111') {
              this.messageEditDialogVisible = false
              this.$message.success('个人资料已保存')
              this.searchFamily()
            } else {
              this.$message.error(response.data.message || '资料未能保存，请检查后重试')
            }
          }).catch(() => { this.$message.error('资料暂时无法保存，请稍后重试') }).then(() => { this.isProfileSaving = false })
        })
      },
      handleAvatarSuccess (res, file) {
        if (!res || res.returnCode !== '1111' || !res.filename) {
          this.$message.error('头像上传未成功，请重试')
          return
        }
        axios.post('restful/family/updateTouPic', { familyTouPic: res.filename, familyPhone: this.familyPhone }, { withCredentials: true }).then((response) => {
          if (response.data.returnCode === '1111') {
            this.$message.success('头像已更新')
            this.searchFamily()
          } else {
            this.$message.error('头像未能更新，请重试')
          }
        }).catch(() => { this.$message.error('头像暂时无法更新，请稍后重试') })
      },
      handleAvatarError () { this.$message.error('头像暂时无法上传，请稍后重试') },
      editPassword (formName) {
        if (this.isPasswordSaving) return
        this.$refs[formName].validate((valid) => {
          if (!valid) return
          this.isPasswordSaving = true
          axios.post('restful/family/updatePassword', {
            familyPhone: this.familyPhone,
            oldPassword: this.passwordEditForm.oldPassword,
            familyPassword: this.passwordEditForm.familyPassword
          }, { withCredentials: true }).then((response) => {
            if (response.data.returnCode === '1111') {
              this.resetForm(formName)
              this.$message.success('修改密码成功')
            } else {
              this.$message.error(response.data.message || '修改密码失败，请检查原密码后重试')
            }
          }).catch(() => {
            this.$message.error('密码暂时无法修改，请稍后重试')
          }).then(() => { this.isPasswordSaving = false })
        })
      },
			handleeditMessage(){
				var _this=this
				_this.messageEditForm.familyId=_this.famiyList.familyId
				_this.messageEditForm.familyName=_this.famiyList.familyName
				_this.messageEditForm.familySex=_this.famiyList.familySex
				_this.messageEditForm.familyAddress=_this.famiyList.familyAddress
				_this.messageEditForm.familyPhone=_this.familyPhone
				this.messageEditDialogVisible = true
        this.$nextTick(() => { this.$refs.profileEditForm.clearValidate() })

			},
			handleeditPassword(){

				var _this=this
				_this.passwordEditForm.familyPhone=_this.familyPhone
				_this.passwordEditDialogVisible = true
			},
			resetForm(formName){
				this.passwordEditDialogVisible = false
				this.$refs[formName].resetFields();
			},
      beforeAvatarUpload(file) {
        const isJPG = ['image/jpeg', 'image/png'].includes(file.type);
        const isLt2M = file.size / 1024 / 1024 < 2;

        if (!isJPG) {
          this.$message.error('请选择 JPG 或 PNG 格式的头像');
        }
        if (!isLt2M) {
          this.$message.error('上传头像图片大小不能超过 2MB!');
        }
        return isJPG && isLt2M;
      }
    }
  }
</script>
