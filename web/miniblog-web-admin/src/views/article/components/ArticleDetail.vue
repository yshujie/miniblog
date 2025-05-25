<template>
  <div class="createPost-container">
    <el-form ref="postForm" :model="postForm" :rules="rules" class="form-container">

      <sticky :z-index="10" :class-name="'sub-navbar '+postForm.status">
        <el-button v-loading="loading" type="default" @click="save">
          保存
        </el-button>
        <el-button v-loading="loading" type="success" @click="publish">
          发布
        </el-button>
        <el-button v-loading="loading" type="warning" @click="unpublish">
          下架
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
                  <el-form-item label-width="60px" label="Author:" class="postInfo-container-item" prop="author">
                    <el-input v-model="postForm.author" placeholder="Please enter the author" />
                  </el-form-item>
                </el-col>

                <el-col :span="10">
                  <el-form-item label-width="120px" label="Tags:" class="postInfo-container-item" prop="tags">
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
                  <el-form-item label-width="60px" label="Module" class="postInfo-container-item" prop="module_code">
                    <el-select v-model="postForm.module_code" placeholder="Module Code" class="module-select" @change="initSections">
                      <el-option v-for="module in modules" :key="module.code" :label="module.title" :value="module.code" />
                    </el-select>
                  </el-form-item>
                </el-col>

                <el-col :span="10">
                  <el-form-item label-width="120px" label="Section" class="postInfo-container-item" prop="section_code">
                    <el-select v-model="postForm.section_code" placeholder="Section Code" class="section-select">
                      <el-option v-for="section in sections" :key="section.code" :label="section.title" :value="section.code" />
                    </el-select>
                  </el-form-item>
                </el-col>
              </el-row>
            </div>
          </el-col>
        </el-row>

        <el-form-item v-show="isEdit" label="Status:" prop="status" style="margin-bottom: 40px;">
          <el-tag
            :key="postForm.status"
            :type="tagType"
            effect="dark"
          >
            {{ postForm.status }}
          </el-tag>
        </el-form-item>

        <el-form-item v-show="! isEdit" label="ExternalLink:" prop="external_link" style="margin-bottom: 40px;">
          <el-input v-model="postForm.external_link" :rows="1" type="textarea" class="article-textarea" autosize placeholder="Please enter the ExternalLink URL" />
        </el-form-item>

        <el-form-item v-show="isEdit" prop="content" style="margin-bottom: 30px;">
          <markdown-editor
            ref="editor"
            v-model="postForm.content"
            height="800px"
            :options="{hideModeSwitch:true,previewStyle:'tab'}"
          />
        </el-form-item>
      </div>
    </el-form>
  </div>
</template>

<script>
import MDinput from '@/components/MDinput'
import Sticky from '@/components/Sticky' // 粘性header组件
import MarkdownEditor from '@/components/MarkdownEditor'
import { fetchArticle, createArticle, publishArticle, updateArticle, unpublishArticle } from '@/api/article'
import { fetchModules } from '@/api/module'
import { fetchSections } from '@/api/section'

const defaultForm = {
  id: '',
  status: '',
  title: '', // 文章题目
  module_code: '', // 模块代码
  section_code: '', // 章节代码
  tags: [], // 文章标签
  external_link: '', // 外部链接
  content: '' // 文章内容
}

export default {
  name: 'ArticleDetail',
  components: { MDinput, Sticky, MarkdownEditor },
  props: {
    isEdit: {
      type: Boolean,
      default: false
    }
  },
  data() {
    const validateRequire = (rule, value, callback) => {
      if (value === null || value === undefined || value.length === 0 || value === '') {
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
      if (value === null || value === undefined || value.length === 0) {
        this.$message({
          message: '标签不能为空',
          type: 'error'
        })
        callback(new Error('标签不能为空'))
      }
      callback()
    }

    const validateExternalLink = (rule, value, callback) => {
      if (this.isEdit) {
        callback()
      } else {
        if (value === null || value === undefined || value.length === 0 || value === '') {
          callback(new Error('外部链接不能为空'))
        }

        // 不是 URL 格式
        if (!value.startsWith('http') || !value.startsWith('https')) {
          callback(new Error('外部链接格式错误'))
        }

        callback()
      }
    }

    return {
      modules: [],
      sections: [],
      postForm: Object.assign({}, defaultForm),
      loading: false,
      userListOptions: [],
      rules: {
        title: [{ validator: validateRequire }],
        tags: [{ validator: validateTags }],
        author: [{ validator: validateRequire }],
        module_code: [{ validator: validateRequire }],
        section_code: [{ validator: validateRequire }],
        external_link: [{ validator: validateExternalLink }]
      },
      tempRoute: {}
    }
  },
  computed: {
    tagType() {
      const statusMap = {
        Draft: 'info',
        Published: 'success',
        Unpublished: 'danger'
      }

      console.log('status', this.postForm.status)
      console.log('tagType', statusMap[this.postForm.status])

      return statusMap[this.postForm.status] || 'info'
    }

  },
  async created() {
    console.log('ArticleDetail created')
    // 初始化 modules
    await this.initModules()

    if (this.isEdit) {
      await this.fetchData()

      await this.initSections()
    }

    // Why need to make a copy of this.$route here?
    // Because if you enter this page and quickly switch tag, may be in the execution of the setTagsViewTitle function, this.$route is no longer pointing to the current page
    // https://github.com/PanJiaChen/vue-element-admin/issues/1221
    this.tempRoute = Object.assign({}, this.$route)
  },
  methods: {
    async fetchData() {
      const id = this.queryArticleId()
      console.log('in fetchData, articleid', id)

      const response = await fetchArticle(id)

      this.postForm = response.article || {}
      console.log('postForm', this.postForm)
    },

    queryArticleId() {
      return this.$route.params && this.$route.params.id
    },

    async initModules() {
      // 清空模块选择
      this.modules = []

      // 获取模块
      const modulesResp = await fetchModules()
      this.modules = modulesResp.modules

      console.log('modules', this.modules)
    },

    async initSections() {
      // 清空章节选择
      this.sections = []

      // 获取章节
      const sectionsResp = await fetchSections(this.postForm.module_code)
      this.sections = sectionsResp.sections
    },

    save() {
      this.$refs.postForm.validate(valid => {
        if (valid) {
          if (this.isEdit) {
            this.updateArticle()
          } else {
            this.createArticle()
          }
        } else {
          return false
        }
      })
    },

    // 创建文章
    async createArticle() {
      this.loading = true

      try {
        // 创建文章
        const resp = await createArticle(this.postForm)
        this.postForm.id = resp.article.id
        this.postForm.status = resp.article.status

        // 保存成功
        this.$message({
          message: '保存成功，文章已创建',
          type: 'success',
          showClose: true,
          duration: 1000,
          onClose: () => {
            this.$router.push({
              path: '/article/edit/' + this.postForm.id
            })
          }
        })
      } catch (error) {
        console.log('error', error)
        this.$message({
          message: '保存失败，错误信息：' + error.message,
          type: 'error',
          showClose: true,
          duration: 3000
        })
      } finally {
        this.loading = false
      }
    },

    // 更新文章
    async updateArticle() {
      this.loading = true

      try {
        // 更新文章
        const resp = await updateArticle(this.postForm)
        this.postForm = resp.article || {}

        this.$message({
          message: '保存成功，文章已更新',
          type: 'success',
          showClose: true,
          duration: 1000
        })

        // 更新数据
        this.fetchData()
      } catch (error) {
        console.log('error', error)
        this.$message({
          message: '保存失败，错误信息：' + error.message,
          type: 'error',
          showClose: true,
          duration: 3000
        })
      } finally {
        this.loading = false
      }
    },

    publish() {
      this.$refs.postForm.validate(valid => {
        if (valid) {
          publishArticle(this.postForm).then(response => {
            this.$message({
              message: '发布成功',
              type: 'success',
              showClose: true,
              duration: 1000
            })

            this.fetchData()
          }).catch(error => {
            console.log('error', error)
            this.$message({
              message: '发布失败，错误信息：' + error.message,
              type: 'error',
              showClose: true,
              duration: 3000
            })
          })
        } else {
          return false
        }
      })
    },

    unpublish() {
      this.$refs.postForm.validate(valid => {
        if (valid) {
          unpublishArticle(this.postForm).then(response => {
            this.$message({
              message: '下架成功',
              type: 'success',
              showClose: true,
              duration: 1000
            })

            this.fetchData()
          }).catch(error => {
            console.log('error', error)
            this.$message({
              message: '下架失败，错误信息：' + error.message,
              type: 'error',
              showClose: true,
              duration: 3000
            })
          })
        } else {
          return false
        }
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
