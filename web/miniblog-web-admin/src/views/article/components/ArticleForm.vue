<template>
  <div class="article-form">
    <el-form ref="formRef" :model="postForm" :rules="rules" label-width="110px" class="form-container">
      <div class="toolbar">
        <el-button type="primary" :loading="loading" @click="handleSave">保存</el-button>
        <el-button type="success" :loading="loading" @click="handlePublish" v-if="isEdit">发布</el-button>
        <el-button type="warning" :loading="loading" @click="handleUnpublish" v-if="isEdit">下架</el-button>
      </div>

      <el-form-item label="标题" prop="title">
        <el-input v-model="postForm.title" placeholder="请输入文章标题" />
      </el-form-item>

      <el-form-item label="作者" prop="author">
        <el-input v-model="postForm.author" placeholder="请输入作者" />
      </el-form-item>

      <el-form-item label="标签" prop="tags">
        <el-select
          v-model="postForm.tags"
          multiple
          filterable
          allow-create
          default-first-option
          placeholder="请输入或选择标签"
          class="tag-select"
        >
          <el-option
            v-for="tag in postForm.tags"
            :key="tag"
            :label="tag"
            :value="tag"
          />
        </el-select>
      </el-form-item>

      <el-form-item label="所属模块" prop="module_code">
        <el-select
          v-model="postForm.module_code"
          placeholder="选择模块"
          class="module-select"
          @change="handleModuleChange"
        >
          <el-option
            v-for="moduleItem in modules"
            :key="moduleItem.code"
            :label="moduleItem.title"
            :value="moduleItem.code"
          />
        </el-select>
      </el-form-item>

      <el-form-item label="所属章节" prop="section_code">
        <el-select
          v-model="postForm.section_code"
          placeholder="选择章节"
          class="section-select"
          :disabled="!sections.length"
        >
          <el-option
            v-for="section in sections"
            :key="section.code"
            :label="section.title"
            :value="section.code"
          />
        </el-select>
      </el-form-item>

      <el-form-item v-if="isEdit" label="当前状态">
        <el-tag :type="statusTagType" effect="dark">{{ postForm.status || '未发布' }}</el-tag>
      </el-form-item>

      <el-form-item label="外链地址" prop="external_link">
        <el-input v-model="postForm.external_link" placeholder="请输入外链地址" />
      </el-form-item>

      <el-form-item label="文章内容" prop="content">
        <el-input
          v-model="postForm.content"
          type="textarea"
          :rows="18"
          placeholder="请输入文章内容（支持 Markdown ）"
        />
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { FormInstance, FormRules } from 'element-plus';
import { ElMessage } from 'element-plus';
import { fetchArticle, createArticle, updateArticle, publishArticle, unpublishArticle } from '@/api/article';
import { fetchModules } from '@/api/module';
import { fetchSections } from '@/api/section';

interface ModuleItem {
  code: string;
  title: string;
}

interface SectionItem {
  code: string;
  title: string;
}

interface ArticleFormState {
  id: string | number;
  status: string;
  title: string;
  author: string;
  module_code: string;
  section_code: string;
  tags: string[];
  external_link: string;
  content: string;
}

const props = defineProps<{ isEdit: boolean }>();

const route = useRoute();
const router = useRouter();

const formRef = ref<FormInstance>();
const loading = ref(false);
const modules = ref<ModuleItem[]>([]);
const sections = ref<SectionItem[]>([]);

const postForm = reactive<ArticleFormState>({
  id: '',
  status: '',
  title: '',
  author: '',
  module_code: '',
  section_code: '',
  tags: [],
  external_link: '',
  content: ''
});

const validateRequired = (message: string) => {
  return (_rule: unknown, value: unknown, callback: (error?: Error) => void) => {
    if (value === undefined || value === null || value === '' || (Array.isArray(value) && value.length === 0)) {
      callback(new Error(message));
    } else {
      callback();
    }
  };
};

const validateExternalLink = (_rule: unknown, value: string, callback: (error?: Error) => void) => {
  if (!value) {
    callback(new Error('外链地址不能为空'));
    return;
  }
  const pattern = /^https?:\/\//i;
  if (!pattern.test(value)) {
    callback(new Error('请输入有效的 http(s) 链接'));
    return;
  }
  callback();
};

const rules: FormRules = {
  title: [{ validator: validateRequired('标题不能为空'), trigger: 'blur' }],
  author: [{ validator: validateRequired('作者不能为空'), trigger: 'blur' }],
  tags: [{ validator: validateRequired('标签不能为空'), trigger: 'change' }],
  module_code: [{ validator: validateRequired('请选择模块'), trigger: 'change' }],
  section_code: [{ validator: validateRequired('请选择章节'), trigger: 'change' }],
  external_link: [{ validator: validateExternalLink, trigger: 'blur' }],
  content: [{ validator: validateRequired('文章内容不能为空'), trigger: 'blur' }]
};

const statusTagType = computed(() => {
  switch (postForm.status) {
    case 'Published':
      return 'success';
    case 'Draft':
      return 'info';
    case 'Unpublished':
      return 'danger';
    default:
      return 'info';
  }
});

const resolveArticleId = () => {
  return route.params?.id as string;
};

const loadModules = async () => {
  try {
    const response = await fetchModules() as any;
    modules.value = response.modules || [];
  } catch (error: any) {
    ElMessage.error(error.message || '加载模块失败');
  }
};

const loadSections = async (moduleCode: string) => {
  if (!moduleCode) {
    sections.value = [];
    postForm.section_code = '';
    return;
  }
  try {
    const response = await fetchSections(moduleCode) as any;
    sections.value = response.sections || [];
  } catch (error: any) {
    ElMessage.error(error.message || '加载章节失败');
  }
};

const loadArticle = async () => {
  if (!props.isEdit) {
    return;
  }
  const articleId = resolveArticleId();
  if (!articleId) {
    ElMessage.error('未找到文章 ID');
    return;
  }
  try {
    const response = await fetchArticle(articleId) as any;
    const article = response.article || {};
    postForm.id = article.id;
    postForm.status = article.status || '';
    postForm.title = article.title || '';
    postForm.author = article.author || '';
    postForm.tags = Array.isArray(article.tags) ? [...article.tags] : [];
    postForm.external_link = article.external_link || '';
    postForm.content = article.content || '';
    postForm.module_code = article.module?.code || '';
    postForm.section_code = article.section?.code || '';

    if (postForm.module_code) {
      await loadSections(postForm.module_code);
    }
  } catch (error: any) {
    ElMessage.error(error.message || '加载文章详情失败');
  }
};

const handleModuleChange = async (moduleCode: string) => {
  await loadSections(moduleCode);
  postForm.section_code = '';
};

const submit = async (handler: () => Promise<void>) => {
  if (!formRef.value) return;
  const valid = await formRef.value.validate();
  if (!valid) return;
  loading.value = true;
  try {
    await handler();
  } finally {
    loading.value = false;
  }
};

const handleSave = async () => {
  await submit(async () => {
    try {
      if (props.isEdit) {
        await updateArticle({ ...postForm });
        ElMessage.success('文章更新成功');
        await loadArticle();
      } else {
        const response = await createArticle({ ...postForm }) as any;
        const newId = response.article?.id;
        ElMessage.success('文章创建成功');
        if (newId) {
          router.replace({ path: `/article/edit/${newId}` });
        }
      }
    } catch (error: any) {
      ElMessage.error(error.message || '保存失败');
      throw error;
    }
  });
};

const handlePublish = async () => {
  await submit(async () => {
    try {
      await publishArticle({ id: postForm.id });
      ElMessage.success('文章发布成功');
      await loadArticle();
    } catch (error: any) {
      ElMessage.error(error.message || '发布失败');
      throw error;
    }
  });
};

const handleUnpublish = async () => {
  await submit(async () => {
    try {
      await unpublishArticle({ id: postForm.id });
      ElMessage.success('文章下架成功');
      await loadArticle();
    } catch (error: any) {
      ElMessage.error(error.message || '下架失败');
      throw error;
    }
  });
};

onMounted(async () => {
  await loadModules();
  if (props.isEdit) {
    await loadArticle();
  }
});
</script>

<style scoped>
.article-form {
  padding: 24px;
  background: #ffffff;
  border-radius: 8px;
}

.toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 24px;
}

.tag-select,
.module-select,
.section-select {
  width: 320px;
}

.form-container {
  max-width: 900px;
}
</style>
