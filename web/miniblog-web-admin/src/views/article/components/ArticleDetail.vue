<template>
  <div class="createPost-container">
    <el-form ref="postForm" :model="postForm" :rules="rules" class="form-container">

      <sticky :z-index="10" :class-name="'sub-navbar '+postForm.status">
        <el-button v-loading="loading" style="margin-left: 10px;" type="success" @click="save">
          保存
        </el-button>
        <el-button v-loading="loading" type="warning" @click="publish">
          发布
        </el-button>
      </sticky>

      <div class="createPost-main-container">
        <el-row>
          <el-col :span="24">
            <el-form-item style="margin-bottom: 40px;" prop="title">
              <MDinput v-model="postForm.title" :maxlength="100" name="name" required>
                Title
              </MDinput>
            </el-form-item>

            <div class="postInfo-container">
              <el-row>
                <el-col :span="8">
                  <el-form-item label-width="60px" label="Author:" class="postInfo-container-item">
                    <el-input v-model="postForm.author" placeholder="Please enter the author" />
                  </el-form-item>
                </el-col>

                <el-col :span="10">
                  <el-form-item label-width="120px" label="Tags:" class="postInfo-container-item">
                    <el-select
                      v-model="postForm.tags"
                      multiple
                      filterable
                      allow-create
                      default-first-option
                      placeholder="文章标签"
                    >
                      <el-option
                        v-for="tag in postForm.tags"
                        :key="tag"
                        :label="tag"
                        :value="tag"
                      />
                    </el-select>

                  </el-form-item>
                </el-col>
              </el-row>
            </div>

            <div class="postInfo-container">
              <el-row>
                <el-col :span="8">
                  <el-form-item label-width="60px" label="Module" class="postInfo-container-item">
                    <el-select v-model="postForm.module_code" placeholder="Module Code" class="module-select" @change="initSections">
                      <el-option v-for="module in modules" :key="module.code" :label="module.title" :value="module.code" />
                    </el-select>
                  </el-form-item>
                </el-col>

                <el-col :span="10">
                  <el-form-item label-width="120px" label="Section" class="postInfo-container-item">
                    <el-select v-model="postForm.section_code" placeholder="Section Code" class="section-select">
                      <el-option v-for="section in sections" :key="section.code" :label="section.title" :value="section.code" />
                    </el-select>
                  </el-form-item>
                </el-col>
              </el-row>
            </div>
          </el-col>
        </el-row>

        <el-form-item style="margin-bottom: 40px;" label="ExternalLink:">
          <el-input v-model="postForm.external_link" :rows="1" type="textarea" class="article-textarea" autosize placeholder="Please enter the ExternalLink URL" />
        </el-form-item>
      </div>
    </el-form>
  </div>
</template>

<script>
import MDinput from '@/components/MDinput'
import Sticky from '@/components/Sticky' // 粘性header组件
import { fetchArticle, createArticle, publishArticle, updateArticle } from '@/api/article'
import { fetchModules } from '@/api/module'
import { fetchSections } from '@/api/section'

const defaultForm = {
  status: 'draft',
  title: '', // 文章题目
  module_code: '', // 模块代码
  section_code: '', // 章节代码
  external_link: '', // 外部链接
  tags: '', // 文章标签
  id: ''
}

export default {
  name: 'ArticleDetail',
  components: { MDinput, Sticky },
  props: {
    isEdit: {
      type: Boolean,
      default: false
    }
  },
  data() {
    const validateRequire = (rule, value, callback) => {
      if (value === '') {
        this.$message({
          message: rule.field + '为必传项',
          type: 'error'
        })
        callback(new Error(rule.field + '为必传项'))
      } else {
        callback()
      }
    }
    const validateTags = (rule, value, callback) => {
      if (value === '') {
        this.$message({
          message: '标签不能为空',
          type: 'error'
        })
        callback(new Error('标签不能为空'))
      }
      callback()
    }
    return {
      modules: [],
      sections: [],
      postForm: Object.assign({}, defaultForm),
      loading: false,
      userListOptions: [],
      rules: {
        title: [{ validator: validateRequire }],
        content: [{ validator: validateRequire }],
        tags: [{ validator: validateTags }]
      },
      tempRoute: {}
    }
  },
  computed: {
    contentShortLength() {
      return this.postForm.content_short.length
    },
    displayTime: {
      // set and get is useful when the data
      // returned by the back end api is different from the front end
      // back end return => "2013-06-25 06:59:25"
      // front end need timestamp => 1372114765000
      get() {
        return (+new Date(this.postForm.display_time))
      },
      set(val) {
        this.postForm.display_time = new Date(val)
      }
    }
  },
  created() {
    console.log('ArticleDetail created')
    // 初始化 modules
    this.initModules()

    if (this.isEdit) {
      const id = this.$route.params && this.$route.params.id
      this.fetchData(id)
    }

    // Why need to make a copy of this.$route here?
    // Because if you enter this page and quickly switch tag, may be in the execution of the setTagsViewTitle function, this.$route is no longer pointing to the current page
    // https://github.com/PanJiaChen/vue-element-admin/issues/1221
    this.tempRoute = Object.assign({}, this.$route)
  },
  methods: {
    fetchData(id) {
      fetchArticle(id).then(response => {
        this.postForm = response.data

        // just for test
        this.postForm.title += `   Article Id:${this.postForm.id}`
        this.postForm.content_short += `   Article Id:${this.postForm.id}`

        // set tagsview title
        this.setTagsViewTitle()

        // set page title
        this.setPageTitle()
      }).catch(err => {
        console.log(err)
      })
    },

    async initModules() {
      // 清空模块选择
      this.postForm.module_code = ''
      this.modules = []

      // 获取模块
      const modulesResp = await fetchModules()
      this.modules = modulesResp.modules

      console.log('modules', this.modules)
    },

    async initSections() {
      // 清空章节选择
      this.postForm.section_code = ''
      this.sections = []

      // 获取章节
      const sectionsResp = await fetchSections(this.postForm.module_code)
      this.sections = sectionsResp.sections
    },

    setTagsViewTitle() {
      const title = 'Edit Article'
      const route = Object.assign({}, this.tempRoute, { title: `${title}-${this.postForm.id}` })
      this.$store.dispatch('tagsView/updateVisitedView', route)
    },
    setPageTitle() {
      const title = 'Edit Article'
      document.title = `${title} - ${this.postForm.id}`
    },

    save() {
      console.log(this.postForm)
      this.$refs.postForm.validate(valid => {
        if (valid) {
          this.loading = true
          this.$notify({
            title: '成功',
            message: '发布文章成功',
            type: 'success',
            duration: 2000
          })
          this.postForm.status = 'published'
          this.loading = false
        } else {
          console.log('error submit!!')
          return false
        }
      }).then(async() => {
        if (this.isEdit) {
          updateArticle(this.postForm)

          this.$message({
            message: '保存成功，文章已更新',
            type: 'success',
            showClose: true,
            duration: 1000
          })
        } else {
          const resp = await createArticle(this.postForm)
          this.postForm.id = resp.article.id

          this.$message({
            message: '保存成功，文章已创建',
            type: 'success',
            showClose: true,
            duration: 1000
          })
        }
      })
    },

    publish() {
      if (this.postForm.title.length === 0) {
        this.$message({
          message: '请填写必要的标题和内容',
          type: 'warning'
        })
        return
      }

      publishArticle(this.postForm).then(response => {
        this.$message({
          message: '发布成功',
          type: 'success',
          showClose: true,
          duration: 1000
        })

        this.postForm.status = 'published'
      })
    }
  }
}
</script>

<style lang="scss" scoped>
@import "~@/styles/mixin.scss";

.createPost-container {
  position: relative;

  .createPost-main-container {
    padding: 40px 45px 20px 50px;

    .postInfo-container {
      position: relative;
      @include clearfix;
      margin-bottom: 10px;

      .postInfo-container-item {
        float: left;
      }
    }
  }

  .word-counter {
    width: 40px;
    position: absolute;
    right: 10px;
    top: 0px;
  }
}

.module-select {
  width: 200px;
  margin-right: 10px;
}
.section-select {
  width: 200px;
}

.article-textarea ::v-deep {
  textarea {
    padding-right: 40px;
    resize: none;
    border: none;
    border-radius: 0px;
    border-bottom: 1px solid #bfcbd9;
  }
}
</style>
